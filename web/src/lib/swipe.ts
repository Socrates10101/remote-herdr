// Edge swipe to go back, for the home-screen app (standalone) where iOS gives
// no browser back gesture. In a normal browser tab the native back gesture
// already works through history, so this stays off to avoid a double back.

const EDGE = 28 // px from the left edge where a swipe may start
const TRIGGER = 80 // px of horizontal travel that commits the back

export function isStandalone(): boolean {
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  )
}

export function watchEdgeSwipe(enabled: () => boolean, onBack: () => void): () => void {
  if (!isStandalone()) return () => {}
  let startX = 0
  let startY = 0
  let tracking = false
  const root = document.documentElement

  const reset = () => {
    tracking = false
    root.style.removeProperty('--swipe-x')
    root.classList.remove('swiping')
  }
  const onStart = (e: TouchEvent) => {
    const t = e.touches[0]
    if (!enabled() || e.touches.length !== 1 || t.clientX > EDGE) return
    tracking = true
    startX = t.clientX
    startY = t.clientY
  }
  const onMove = (e: TouchEvent) => {
    if (!tracking) return
    const t = e.touches[0]
    const dx = t.clientX - startX
    const dy = Math.abs(t.clientY - startY)
    if (dy > 30 && dy > dx) return reset() // vertical scroll wins
    if (dx > 0) {
      root.classList.add('swiping')
      root.style.setProperty('--swipe-x', `${Math.min(dx, window.innerWidth)}px`)
    }
  }
  const onEnd = (e: TouchEvent) => {
    if (!tracking) return
    const dx = (e.changedTouches[0]?.clientX ?? startX) - startX
    reset()
    if (dx > TRIGGER) onBack()
  }
  document.addEventListener('touchstart', onStart, { passive: true })
  document.addEventListener('touchmove', onMove, { passive: true })
  document.addEventListener('touchend', onEnd, { passive: true })
  document.addEventListener('touchcancel', reset, { passive: true })
  return () => {
    document.removeEventListener('touchstart', onStart)
    document.removeEventListener('touchmove', onMove)
    document.removeEventListener('touchend', onEnd)
    document.removeEventListener('touchcancel', reset)
  }
}
