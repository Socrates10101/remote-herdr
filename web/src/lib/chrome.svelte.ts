// Shared chrome state for phones.
// - Header: slides away on any scroll so reading gets the screen.
// - Dock (composer): hides only after a deliberate downward scroll and comes
//   back on the slightest upward scroll, so it is always easy to reach.
// Both return at the top or bottom of the page, and a tap on the content
// toggles them. Wide screens keep both visible via CSS.

export const chrome = $state({ hidden: false, dockHidden: false })

const HEADER_STEP = 8 // px of movement (either way) that hides the header
const DOCK_HIDE = 24 // px of continuous downward scroll that hides the dock
const DOCK_SHOW = 8 // px of upward scroll that brings the dock back

let lastY = 0
let downRun = 0
let upRun = 0
let ticking = false

function update() {
  ticking = false
  const y = window.scrollY
  const max = document.documentElement.scrollHeight - window.innerHeight
  const delta = y - lastY
  lastY = y
  if (y < 48 || y > max - 24) {
    chrome.hidden = false
    chrome.dockHidden = false
    downRun = upRun = 0
    return
  }
  if (Math.abs(delta) >= HEADER_STEP) chrome.hidden = true
  if (delta > 0) {
    downRun += delta
    upRun = 0
    if (downRun >= DOCK_HIDE) chrome.dockHidden = true
  } else if (delta < 0) {
    upRun -= delta
    downRun = 0
    if (upRun >= DOCK_SHOW) chrome.dockHidden = false
  }
}

// Taps on links, buttons, form fields, code and expandable rows keep their
// own meaning; anything else toggles the chrome.
const INTERACTIVE = 'a, button, input, textarea, select, summary, details, pre, code, img, .dock, .topbar, .copy'

export function watchScroll(): () => void {
  lastY = window.scrollY
  const onScroll = () => {
    if (!ticking) {
      ticking = true
      requestAnimationFrame(update)
    }
  }
  const onClick = (e: MouseEvent) => {
    const target = e.target as Element | null
    if (!target || target.closest(INTERACTIVE)) return
    if (window.getSelection()?.toString()) return // finishing a text selection
    const hide = !(chrome.hidden || chrome.dockHidden)
    chrome.hidden = hide
    chrome.dockHidden = hide
  }
  window.addEventListener('scroll', onScroll, { passive: true })
  document.addEventListener('click', onClick)
  return () => {
    window.removeEventListener('scroll', onScroll)
    document.removeEventListener('click', onClick)
  }
}

// Scroll reports from a framed preview (see PreviewViewer). Down hides the
// top bar and the composer together; up, the top, or the bottom brings
// them back.
const FRAME_STEP = 6
let frameY = 0

export function followFrameScroll(y: number, max: number) {
  const delta = y - frameY
  frameY = y
  if (y < 48 || (max > 0 && y > max - 24)) {
    revealChrome()
  } else if (delta > FRAME_STEP) {
    chrome.hidden = true
    chrome.dockHidden = true
  } else if (delta < -FRAME_STEP) {
    revealChrome()
  }
}

export function resetFrameScroll() {
  frameY = 0
  showChrome()
}

// While a preview is shown the bars are rarely needed: they hide after a
// short idle, unless the user is typing, pointing at them, or has the
// picker open. Showing them again (edge strip, scroll up) re-arms it.
const IDLE_MS = 2500
let idleTimer: ReturnType<typeof setTimeout> | undefined

function holdChrome(): boolean {
  const active = document.activeElement
  if (active && active !== document.body && active.closest('.dock, .topbar, .pmenu')) return true
  if (document.querySelector('.pmenu')) return true
  return !!document.querySelector('.topbar:hover, .dock:hover')
}

export function armIdleHide() {
  clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    if (holdChrome()) return armIdleHide()
    chrome.hidden = true
    chrome.dockHidden = true
  }, IDLE_MS)
}

export function stopIdleHide() {
  clearTimeout(idleTimer)
  idleTimer = undefined
}

// revealChrome shows the bars and starts the idle countdown again.
export function revealChrome() {
  showChrome()
  if (idleTimer !== undefined) armIdleHide()
}

export function showChrome() {
  chrome.hidden = false
  chrome.dockHidden = false
}
