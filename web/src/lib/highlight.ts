// highlight.js core with only the languages we expect in agent sessions.
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import css from 'highlight.js/lib/languages/css'
import diff from 'highlight.js/lib/languages/diff'
import dockerfile from 'highlight.js/lib/languages/dockerfile'
import go from 'highlight.js/lib/languages/go'
import ini from 'highlight.js/lib/languages/ini'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import latex from 'highlight.js/lib/languages/latex'
import markdown from 'highlight.js/lib/languages/markdown'
import nix from 'highlight.js/lib/languages/nix'
import python from 'highlight.js/lib/languages/python'
import rust from 'highlight.js/lib/languages/rust'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'

const langs: Record<string, Parameters<typeof hljs.registerLanguage>[1]> = {
  bash, css, diff, dockerfile, go, ini, javascript, json, latex, markdown, nix, python, rust, sql, typescript, xml, yaml,
}
for (const [name, def] of Object.entries(langs)) hljs.registerLanguage(name, def)
hljs.registerAliases(['sh', 'zsh', 'shell', 'console'], { languageName: 'bash' })
hljs.registerAliases(['tex', 'bib'], { languageName: 'latex' })
hljs.registerAliases(['toml'], { languageName: 'ini' })
hljs.registerAliases(['ts', 'tsx'], { languageName: 'typescript' })
hljs.registerAliases(['js', 'jsx', 'mjs'], { languageName: 'javascript' })
hljs.registerAliases(['html', 'svg'], { languageName: 'xml' })
hljs.registerAliases(['md'], { languageName: 'markdown' })
hljs.registerAliases(['py'], { languageName: 'python' })
hljs.registerAliases(['yml'], { languageName: 'yaml' })
hljs.registerAliases(['patch'], { languageName: 'diff' })

export default hljs

const extLang: Record<string, string> = {
  sh: 'bash', zsh: 'bash', bash: 'bash', css: 'css', diff: 'diff', patch: 'diff', go: 'go',
  ini: 'ini', toml: 'ini', js: 'javascript', mjs: 'javascript', jsx: 'javascript', json: 'json',
  tex: 'latex', sty: 'latex', cls: 'latex', bib: 'latex', md: 'markdown', nix: 'nix', py: 'python',
  rs: 'rust', sql: 'sql', ts: 'typescript', tsx: 'typescript', html: 'xml', xml: 'xml', svg: 'xml',
  yaml: 'yaml', yml: 'yaml',
}

export function langForPath(path: string): string {
  const name = path.split('/').pop() ?? ''
  if (name === 'Dockerfile') return 'dockerfile'
  const ext = name.includes('.') ? name.split('.').pop()!.toLowerCase() : ''
  return extLang[ext] ?? ''
}
