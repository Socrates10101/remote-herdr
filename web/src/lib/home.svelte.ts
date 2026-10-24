// The server's home directory, so paths can be shown as "~/…" on any OS.
// Reactive: lists rendered before /api/me answers update when it does.

const home = $state({ dir: '' })

export function setHome(dir: string) {
  home.dir = dir.replace(/\/+$/, '')
}

export function shortPath(path: string): string {
  const dir = home.dir
  if (dir && (path === dir || path.startsWith(dir + '/'))) return '~' + path.slice(dir.length)
  return path
}
