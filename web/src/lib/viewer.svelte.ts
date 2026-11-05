// What the session's preview tab shows: the session's preview folders and
// the one selected. Reset when another session is opened.

import { api, refKey, type Agent, type Preview } from './api'

// Every preview folder, for the sidebar while a preview is shown.
export const allPreviews = $state({ entries: [] as Preview[], loaded: false })

export async function loadAllPreviews(): Promise<void> {
  const r = await api.previews()
  allPreviews.entries = r.entries
  allPreviews.loaded = true
}

export const sessionPreviews = $state({
  key: '', // refKey of the session the entries belong to
  entries: [] as Preview[],
  loaded: false,
  url: '', // selected preview
  menu: false, // the picker under the tab is open
  // A preview of an ended or unknown session shown in this session's frame.
  adhoc: null as Preview | null,
})

export function resetSessionPreviews(key: string) {
  sessionPreviews.key = key
  sessionPreviews.entries = []
  sessionPreviews.loaded = false
  sessionPreviews.url = ''
  sessionPreviews.menu = false
  sessionPreviews.adhoc = null
}

export async function loadSessionPreviews(a: Agent): Promise<void> {
  const key = refKey(a)
  const r = await api.agentPreviews(a)
  if (sessionPreviews.key !== key) return
  sessionPreviews.entries = r.entries
  sessionPreviews.loaded = true
  const u = sessionPreviews.url
  if (u && !r.entries.some((p) => p.url === u) && sessionPreviews.adhoc?.url !== u) sessionPreviews.url = ''
}

export function selectedPreview(): Preview | undefined {
  const u = sessionPreviews.url
  const own = sessionPreviews.entries.find((p) => p.url === u)
  if (own || sessionPreviews.adhoc?.url !== u) return own
  // Prefer the fresh copy from the full list (its modified time moves on update).
  return allPreviews.entries.find((p) => p.url === u) ?? sessionPreviews.adhoc
}
