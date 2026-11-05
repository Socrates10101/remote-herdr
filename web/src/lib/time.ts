// ago formats a timestamp as a short Japanese relative time.
export function ago(iso: string, now = Date.now()): string {
  const t = Date.parse(iso)
  if (Number.isNaN(t)) return ''
  const m = Math.floor((now - t) / 60000)
  if (m < 1) return 'たった今'
  if (m < 60) return `${m} 分前`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h} 時間前`
  const d = Math.floor(h / 24)
  if (d < 7) return `${d} 日前`
  const dt = new Date(t)
  return `${dt.getMonth() + 1}/${dt.getDate()}`
}
