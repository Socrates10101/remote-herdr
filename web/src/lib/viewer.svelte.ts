// The in-app preview viewer. Opening pushes a history entry so the browser's
// back (and the viewer's own back button) closes it without leaving the page
// underneath.

export const viewer = $state({ url: '', title: '' })

export function openPreview(url: string, title: string) {
  if (!url.startsWith('https://')) return
  viewer.url = url
  viewer.title = title
  history.pushState({ ...(history.state ?? {}), preview: url }, '', location.hash || '#')
}

export function closePreview() {
  if (!viewer.url) return
  if (history.state?.preview) history.back() // popstate clears the viewer
  else viewer.url = ''
}

// onPopPreview handles popstate; true when it only closed the viewer.
export function onPopPreview(): boolean {
  if (viewer.url && !history.state?.preview) {
    viewer.url = ''
    return true
  }
  return false
}
