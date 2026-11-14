// markdown-it plugin: TeX math via KaTeX.
//
// Recognised forms (the ones Claude emits):
//   inline   $…$   \(…\)
//   display  $$…$$ \[…\]  and a bare \begin{env}…\end{env} at line start
// Display math may span lines, and may start on the line right after text
// without a blank line in between (a block rule that terminates paragraphs).
// A display block whose closing delimiter is followed by text on the same
// line is left to the inline rule, so that text is kept.
//
// Rendering stays within the safety model of markdown.ts: KaTeX is called
// with the caller's options (trust:false there), and the result still goes
// through DOMPurify afterwards.

import type { MarkdownIt, StateBlock, StateInline, Token } from 'markdown-it'
import katex, { type KatexOptions } from 'katex'

export interface MathOptions {
  katexOptions?: KatexOptions
}

// Environments KaTeX renders on their own in display mode.
const ENVS = new Set([
  'align', 'align*', 'alignat', 'alignat*', 'aligned', 'alignedat',
  'equation', 'equation*', 'gather', 'gather*', 'gathered',
  'cases', 'rcases', 'dcases', 'split', 'CD',
  'matrix', 'pmatrix', 'bmatrix', 'Bmatrix', 'vmatrix', 'Vmatrix', 'smallmatrix', 'array', 'darray',
])

function isDigit(c: number): boolean {
  return c >= 0x30 && c <= 0x39
}

// Find `close` in `src` at or after `from`; -1 if absent.
function indexOf(src: string, close: string, from: number): number {
  return src.indexOf(close, from)
}

export default function mathPlugin(md: MarkdownIt, options: MathOptions = {}): void {
  const katexOptions = options.katexOptions ?? {}

  function render(tex: string, displayMode: boolean): string {
    try {
      return katex.renderToString(tex, { ...katexOptions, displayMode })
    } catch (err) {
      const msg = err instanceof Error ? err.message : String(err)
      return `<code class="math-error" title="${md.utils.escapeHtml(msg)}">${md.utils.escapeHtml(tex)}</code>`
    }
  }

  // Inline rule: runs before `escape` so `\(` and `\[` are not unescaped,
  // and before `emphasis` so `_` and `*` inside math are left alone.
  function inline(state: StateInline, silent: boolean): boolean {
    const src = state.src
    const pos = state.pos
    const max = state.posMax
    const ch = src.charCodeAt(pos)
    let open: string
    let close: string
    let display: boolean
    if (ch === 0x5c /* \ */) {
      const next = src.charCodeAt(pos + 1)
      if (next === 0x28 /* ( */) { open = '\\('; close = '\\)'; display = false }
      else if (next === 0x5b /* [ */) { open = '\\['; close = '\\]'; display = true }
      else return false
    } else if (ch === 0x24 /* $ */) {
      if (src.charCodeAt(pos + 1) === 0x24) { open = '$$'; close = '$$'; display = true }
      else {
        // Single dollar: not after a backslash or a digit, content starts and
        // ends with non-space, no newline inside, no digit after the closer.
        // Keeps `$5 and $10` and `\$` as text.
        const prev = pos > 0 ? src.charCodeAt(pos - 1) : 0
        if (prev === 0x5c || isDigit(prev)) return false
        open = '$'; close = '$'; display = false
      }
    } else {
      return false
    }

    const start = pos + open.length
    let end = -1
    if (open === '$') {
      if (start >= max || /\s/.test(src[start]!)) return false
      let i = start
      for (;;) {
        i = indexOf(src, '$', i)
        if (i < 0 || i >= max) return false
        const before = src.charCodeAt(i - 1)
        if (before === 0x5c || /\s/.test(src[i - 1]!)) { i += 1; continue } // `\$` or `… $`: not a closer
        if (isDigit(src.charCodeAt(i + 1))) return false // `$x$1`: treat as text
        end = i
        break
      }
      if (src.slice(start, end).includes('\n')) return false
    } else {
      end = indexOf(src, close, start)
      if (end < 0 || end > max) return false
      if (open === '$$' && end === start) return false
    }

    const content = src.slice(start, end)
    if (content.trim() === '') return false
    if (!silent) {
      const token = state.push(display ? 'math_display' : 'math_inline', 'math', 0)
      token.content = content
      token.markup = open
    }
    state.pos = end + close.length
    return true
  }

  // Block rule: `$$`, `\[` or `\begin{env}` at the start of a line, closed by
  // `$$`, `\]` or `\end{env}` at the end of a (possibly later) line.
  function block(state: StateBlock, startLine: number, endLine: number, silent: boolean): boolean {
    if (state.sCount[startLine]! - state.blkIndent >= 4) return false // indented code
    const lineText = (i: number) => state.src.slice(state.bMarks[i]! + state.tShift[i]!, state.eMarks[i]!)
    const first = lineText(startLine)
    let open: string
    let close: string
    let keepDelims = false
    if (first.startsWith('$$')) { open = '$$'; close = '$$' }
    else if (first.startsWith('\\[')) { open = '\\['; close = '\\]' }
    else if (first.startsWith('\\begin{')) {
      const m = /^\\begin\{([A-Za-z*]+)\}/.exec(first)
      if (!m || !ENVS.has(m[1]!)) return false
      open = m[0]; close = `\\end{${m[1]}}`; keepDelims = true
    } else {
      return false
    }

    // Locate the closing delimiter; it must end its line (modulo whitespace).
    let line = startLine
    let from = open.length
    let endIdx = -1
    for (; line < endLine; line++) {
      const t = lineText(line)
      const i = t.indexOf(close, from)
      if (i >= 0) {
        if (t.slice(i + close.length).trim() !== '') return false // text after closer: inline rule
        endIdx = i
        break
      }
      from = 0
    }
    if (endIdx < 0) return false
    if (silent) return true

    const lines: string[] = []
    for (let i = startLine; i <= line; i++) {
      let t = lineText(i)
      if (i === line) t = t.slice(0, endIdx + (keepDelims ? close.length : 0))
      if (i === startLine) t = keepDelims ? t : t.slice(open.length)
      lines.push(t)
    }
    const content = lines.join('\n')
    if (content.trim() === '') return false

    const token = state.push('math_block', 'math', 0)
    token.block = true
    token.content = content
    token.markup = open
    token.map = [startLine, line + 1]
    state.line = line + 1
    return true
  }

  md.inline.ruler.before('escape', 'math_inline', inline)
  md.block.ruler.before('fence', 'math_block', block, { alt: ['paragraph', 'reference', 'blockquote', 'list'] })

  md.renderer.rules.math_inline = (tokens: Token[], idx: number) => render(tokens[idx]!.content, false)
  md.renderer.rules.math_display = (tokens: Token[], idx: number) => render(tokens[idx]!.content, true)
  md.renderer.rules.math_block = (tokens: Token[], idx: number) => render(tokens[idx]!.content, true) + '\n'
}
