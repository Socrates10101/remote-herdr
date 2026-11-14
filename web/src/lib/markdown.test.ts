import { describe, expect, it } from 'vitest'
import { renderMarkdown, type RenderContext } from './markdown'

const ctx: RenderContext = { ref: { host: 'h', paneId: 'p' }, baseDir: '', cwd: '/w' }
const render = (src: string) => renderMarkdown(src, ctx)
const count = (html: string, re: RegExp) => (html.match(re) ?? []).length
const katex = (html: string) => count(html, /class="katex"/g)
const display = (html: string) => count(html, /class="katex-display"/g)
const text = (html: string) =>
  html
    .replace(/<span class="katex-mathml">.*?<\/span><span class="katex-html"/gs, '<span class="katex-html"')
    .replace(/<[^>]+>/g, '')

describe('inline math', () => {
  it('renders \\(…\\) and $…$', () => {
    const html = render('面積 \\(\\pi r^2\\) と $2\\pi r$')
    expect(katex(html)).toBe(2)
    expect(display(html)).toBe(0)
    expect(html).not.toContain('katex-error')
  })
  it('renders \\(…\\) across a line break', () => {
    const html = render('値は \\(x^2 +\ny_1\\) です。')
    expect(katex(html)).toBe(1)
    expect(text(html)).toContain('です。')
  })
  it('leaves money and shell variables alone', () => {
    expect(katex(render('costs $5 and $10 total'))).toBe(0)
    expect(katex(render('価格は \\$5 です'))).toBe(0)
    expect(katex(render('$x$1 and $y$ 1'))).toBe(0)
    expect(katex(render('`$HOME` と `\\(x\\)` はそのまま'))).toBe(0)
    expect(text(render('`$HOME` と `\\(x\\)` はそのまま'))).toContain('\\(x\\)')
  })
  it('keeps _ and * inside math out of emphasis', () => {
    const html = render('$x_1$ and $y_1$ and _em_')
    expect(katex(html)).toBe(2)
    expect(html).toContain('<em>em</em>')
  })
})

describe('display math', () => {
  it('renders \\[…\\] on the line right after text', () => {
    const html = render('式は次の通り:\n\\[\nE = mc^2\n\\]\n次の文。')
    expect(display(html)).toBe(1)
    expect(text(html)).toContain('次の文。')
    expect(text(html)).not.toContain('[')
  })
  it('renders $$…$$ on the line right after text', () => {
    const html = render('式:\n$$\n\\int_0^1 x\\,dx\n$$\n次。')
    expect(display(html)).toBe(1)
    expect(text(html)).toContain('次。')
  })
  it('keeps text after the closing delimiter', () => {
    const html = render('\\[ x = 2 \\] となる。')
    expect(display(html)).toBe(1)
    expect(text(html)).toContain('となる。')
  })
  it('renders \\[…\\] in the middle of a line', () => {
    const html = render('よって \\[ x = 2 \\] となる。')
    expect(display(html)).toBe(1)
    expect(text(html)).toContain('よって')
    expect(text(html)).toContain('となる。')
  })
  it('renders inside list items and blockquotes', () => {
    const list = render('1. 式:\n   \\[\n   a = b\n   \\]\n2. 次')
    expect(display(list)).toBe(1)
    expect(count(list, /<li>/g)).toBe(2)
    const quote = render('> 式\n> \\[\n> a = b\n> \\]')
    expect(display(quote)).toBe(1)
    expect(quote).not.toContain('katex-error')
  })
  it('renders aligned environments, including a line starting with -', () => {
    const html = render('text:\n\\[\n\\begin{aligned}\na &= b \\\\\n- c &= d\n\\end{aligned}\n\\]')
    expect(display(html)).toBe(1)
    expect(html).not.toContain('<ul>')
    expect(html).not.toContain('katex-error')
  })
  it('renders a bare \\begin{align}…\\end{align}', () => {
    const html = render('\\begin{align}\na &= b \\\\\nc &= d\n\\end{align}')
    expect(display(html)).toBe(1)
    expect(html).not.toContain('katex-error')
  })
  it('does not touch fences or unknown environments', () => {
    expect(katex(render('```tex\n\\[ x \\]\n```'))).toBe(0)
    expect(katex(render('\\begin{document}\nhi\n\\end{document}'))).toBe(0)
  })
})

describe('safety', () => {
  it('keeps KaTeX MathML (semantics/annotation) and no stray source text', () => {
    const html = render('$x^2$')
    expect(html).toContain('<semantics>')
    expect(html).toContain('<annotation encoding="application/x-tex">x^2</annotation>')
    expect(html).not.toMatch(/<\/mrow>x\^2<\/math>/)
  })
  it('does not let math emit links, images or raw HTML', () => {
    const html = render('$\\href{javascript:alert(1)}{x}$ \\(\\includegraphics{x}\\) $\\htmlClass{a}{b}$')
    expect(html).not.toContain('<a ')
    expect(html).not.toContain('<img')
    expect(html).not.toMatch(/\bhref=/)
    expect(html).not.toContain('<script')
  })
  it('shows a bad formula as an error, not raw HTML', () => {
    const html = render('$\\frac{<b>$')
    expect(html).toContain('katex-error')
    expect(html).not.toContain('<b>')
  })
})
