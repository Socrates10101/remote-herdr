// What the session's preview tab shows: the session's preview folders and
// the one selected. Reset when another session is opened.

import { api, refKey, type Agent, type Preview } from './api'

export const sessionPreviews = $state({
  key: '', // refKey of the session the entries belong to
  entries: [] as Preview[],
  loaded: false,
  url: '', // selected preview
  menu: false, // the picker under the tab is open
})

export function resetSessionPreviews(key: string) {
  sessionPreviews.key = key
  sessionPreviews.entries = []
  sessionPreviews.loaded = false
  sessionPreviews.url = ''
  sessionPreviews.menu = false
}

export async function loadSessionPreviews(a: Agent): Promise<void> {
  const key = refKey(a)
  const r = await api.agentPreviews(a)
  if (sessionPreviews.key !== key) return
  sessionPreviews.entries = r.entries
  sessionPreviews.loaded = true
  if (sessionPreviews.url && !r.entries.some((p) => p.url === sessionPreviews.url)) sessionPreviews.url = ''
}

export function selectedPreview(): Preview | undefined {
  return sessionPreviews.entries.find((p) => p.url === sessionPreviews.url)
}
