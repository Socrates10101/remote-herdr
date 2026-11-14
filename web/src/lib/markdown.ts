// Markdown + math + code rendering.
//
// Safety model: markdown-it runs with html:false (raw HTML in the source is
// escaped), KaTeX runs with trust:false, and the final HTML is passed
// through DOMPurify with a hook that only keeps same-origin API images and
// http(s)/relative links. The strict CSP is the last line of defence.

import MarkdownIt from 'markdown-it'
import mathPlugin from './math'
import DOMPurify from 'dompurify'
import hljs from './highlight'
import { fileURL, type AgentRef } from './api'
import { joinPath } from './paths'

export interface RenderContext {
  ref: AgentRef
  // Directory (relative to the agent cwd) that relative image paths resolve against.
  baseDir: string
  // Absolute agent cwd, so absolute paths inside it can be shown.
  cwd: string
}

function highlight(code: string, lang: string): string {
  const language = lang && hljs.getLanguage(lang) ? lang : ''
  if (!language) return '' // markdown-it escapes it
  try {
    return hljs.highlight(code, { language, ignoreIllegals: true }).value
  } catch {
    return ''
  }
}

function createMd() {
  const md = new MarkdownIt({ html: false, linkify: true, typographer: false, breaks: false, highlight })
  md.use(mathPlugin, {
    katexOptions: { throwOnError: false, trust: false, strict: 'ignore', output: 'htmlAndMathml' },
  })
  // Code fences get a header with the language and a copy control
  // (a span with role=button: DOMPurify drops real <button>s).
  md.renderer.rules.fence = (tokens, idx) => {
    const t = tokens[idx]
    const lang = t.info.trim().split(/\s+/)[0] ?? ''
    const body = highlight(t.content, lang) || md.utils.escapeHtml(t.content)
    const label = md.utils.escapeHtml(lang || 'text')
    return (
      `<div class="code"><div class="code-head"><span>${label}</span>` +
      `<span class="copy" role="button" tabindex="0">コピー</span></div>` +
      `<pre class="hljs"><code>${body}</code></pre></div>`
    )
  }
  return md
}

const md = createMd()

// Map an image src from markdown to a same-origin file URL, or '' to drop it.
function imageSrc(src: string, ctx: RenderContext): string {
  if (/^data:image\/(png|jpeg|gif|webp);base64,/i.test(src)) return src
  if (/^[a-z][a-z0-9+.-]*:/i.test(src) || src.startsWith('//')) return '' // remote: blocked
  let rel: string
  if (src.startsWith('/')) {
    const root = ctx.cwd.endsWith('/') ? ctx.cwd : ctx.cwd + '/'
    if (!src.startsWith(root)) return ''
    rel = src.slice(root.length)
  } else {
    rel = joinPath(ctx.baseDir, src)
  }
  if (rel === '' || rel.startsWith('..')) return ''
  return fileURL(ctx.ref, rel)
}

let currentCtx: RenderContext | null = null

DOMPurify.addHook('afterSanitizeAttributes', (node) => {
  if (node.tagName === 'A') {
    const href = node.getAttribute('href') ?? ''
    if (!/^https?:\/\//i.test(href)) {
      node.removeAttribute('href')
    } else {
      node.setAttribute('target', '_blank')
      node.setAttribute('rel', 'noopener noreferrer')
    }
  }
  if (node.tagName === 'IMG') {
    const src = node.getAttribute('src') ?? ''
    const mapped = currentCtx ? imageSrc(src, currentCtx) : ''
    if (mapped) {
      node.setAttribute('src', mapped)
      node.setAttribute('loading', 'lazy')
    } else {
      node.removeAttribute('src')
      node.setAttribute('alt', `[image blocked: ${src}]`)
    }
  }
})

const purifyConfig = {
  USE_PROFILES: { html: true, mathMl: true, svg: true },
  // KaTeX wraps its MathML in <semantics> with a text-only <annotation>
  // holding the TeX source. DOMPurify drops both by default and leaves the
  // source as a stray text node, so allow these two (text-only) elements.
  // annotation-xml (an HTML integration point) stays forbidden.
  ADD_TAGS: ['semantics', 'annotation'],
  ADD_ATTR: ['encoding'],
  FORBID_TAGS: ['style', 'form', 'input', 'button', 'textarea', 'select', 'iframe', 'object', 'embed'],
  FORBID_ATTR: ['srcset'],
}

export function renderMarkdown(source: string, ctx: RenderContext): string {
  currentCtx = ctx
  try {
    return DOMPurify.sanitize(md.render(source), purifyConfig) as string
  } finally {
    currentCtx = null
  }
}

export function renderCode(source: string, lang: string): string {
  const html = highlight(source, lang)
  const body = html || escapeHtml(source)
  return DOMPurify.sanitize(`<pre class="hljs"><code>${body}</code></pre>`, purifyConfig) as string
}

export function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c]!)
}
