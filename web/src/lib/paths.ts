// Relative path helpers. Paths are relative to the agent's cwd and use "/".

// joinPath resolves rel against base. It returns a path that may start
// with ".." when it escapes the root; callers must refuse those.
export function joinPath(base: string, rel: string): string {
  const parts = (base ? base.split('/') : []).concat(rel.split('/'))
  const out: string[] = []
  for (const p of parts) {
    if (p === '' || p === '.') continue
    if (p === '..') {
      if (out.length === 0 || out[out.length - 1] === '..') out.push('..')
      else out.pop()
    } else {
      out.push(p)
    }
  }
  return out.join('/')
}

export function dirname(path: string): string {
  const i = path.lastIndexOf('/')
  return i < 0 ? '' : path.slice(0, i)
}

