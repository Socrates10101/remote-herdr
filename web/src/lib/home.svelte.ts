// Server facts from /api/me: the home directory, so paths can be shown as
// "~/…" on any OS, and whether "!" shell prompts are allowed.
// Reactive: views rendered before /api/me answers update when it does.

const home = $state({ dir: '', allowShell: false })

export function setHome(dir: string) {
  home.dir = dir.replace(/\/+$/, '')
}

export function setAllowShell(v: boolean) {
  home.allowShell = v
}

export function shellAllowed(): boolean {
  return home.allowShell
}

export function shortPath(path: string): string {
  const dir = home.dir
  if (dir && (path === dir || path.startsWith(dir + '/'))) return '~' + path.slice(dir.length)
  return path
}
