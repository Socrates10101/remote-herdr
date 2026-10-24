// Thin client for the remote-herdr API. Every POST carries the CSRF header
// and a JSON content type, which the server requires.

export type Status = 'idle' | 'working' | 'blocked' | 'done' | 'unknown' | string

// AgentRef identifies a session across hosts. Today there is one host; a
// future hub will route by host as well (see ARCHITECTURE.md).
export interface AgentRef {
  host: string
  paneId: string
}

export const refKey = (r: AgentRef) => `${r.host}/${r.paneId}`

export interface Agent {
  host: string
  paneId: string
  workspaceId: string
  tabId: string
  agent: string
  sessionId: string
  status: Status
  title: string
  cwd: string
  focused: boolean
}

export interface Block {
  kind: 'text' | 'tool_use' | 'tool_result' | 'image'
  text?: string
  name?: string
  input?: string
  isError?: boolean
  mediaType?: string
  data?: string
  truncated?: boolean
}

export interface Message {
  uuid: string
  timestamp: string
  role: 'user' | 'assistant'
  blocks: Block[]
}

export interface Entry {
  name: string
  dir: boolean
  size: number
  kind: 'dir' | 'image' | 'pdf' | 'text' | 'other'
}

export interface DialogItem {
  number?: string
  label: string
  description?: string
  checked?: boolean
  freeText?: boolean
}

export interface Dialog {
  context: string[]
  items: DialogItem[]
  cursor: number
}

export interface State {
  agents: Agent[]
  sizes: Record<string, number>
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message)
  }
}

async function check(res: Response): Promise<Response> {
  if (res.ok) return res
  let msg = res.statusText
  try {
    const body = await res.json()
    if (body && typeof body.error === 'string') msg = body.error
  } catch {
    /* not JSON */
  }
  throw new ApiError(res.status, msg)
}

async function get<T>(path: string): Promise<T> {
  const res = await check(await fetch(path, { credentials: 'same-origin' }))
  return res.json() as Promise<T>
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await check(
    await fetch(path, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json', 'X-Remote-Herdr': '1' },
      body: JSON.stringify(body),
    }),
  )
  return res.json() as Promise<T>
}

// agentBase is the single place that maps a session to its API path.
// With one host every agent is local; a hub would prefix /api/hosts/{host}.
function agentBase(r: AgentRef): string {
  return `/api/agents/${encodeURIComponent(r.paneId)}`
}

export const api = {
  me: () => get<{ user: string; host: string; home: string }>('/api/me'),
  agents: () => get<{ agents: Agent[] }>('/api/agents'),
  screen: (r: AgentRef, source = 'visible') =>
    get<{ text: string; status: Status }>(`${agentBase(r)}/screen?source=${source}`),
  transcript: (r: AgentRef) => get<{ messages: Message[]; size: number }>(`${agentBase(r)}/transcript`),
  files: (r: AgentRef, path: string) =>
    get<{ entries: Entry[] }>(`${agentBase(r)}/files?path=${encodeURIComponent(path)}`),
  fileText: async (r: AgentRef, path: string) => {
    const res = await check(await fetch(fileURL(r, path), { credentials: 'same-origin' }))
    return res.text()
  },
  prompt: (r: AgentRef, text: string) => post<{ ok: boolean }>(`${agentBase(r)}/prompt`, { text }),
  decision: (r: AgentRef, decision: 'approve' | 'deny') =>
    post<{ ok: boolean }>(`${agentBase(r)}/decision`, { decision }),
  dialog: (r: AgentRef) => get<{ dialog: Dialog | null }>(`${agentBase(r)}/dialog`),
  choose: (r: AgentRef, index: number, label: string) =>
    post<{ ok: boolean }>(`${agentBase(r)}/choose`, { index, label }),
  interrupt: (r: AgentRef) => post<{ ok: boolean }>(`${agentBase(r)}/interrupt`, {}),
}

export function fileURL(r: AgentRef, path: string): string {
  return `${agentBase(r)}/file?path=${encodeURIComponent(path)}`
}

// subscribe streams state updates; the browser reconnects automatically.
export function subscribe(onState: (s: State) => void, onError: (down: boolean) => void): () => void {
  const es = new EventSource('/api/events')
  es.addEventListener('state', (ev) => {
    onError(false)
    onState(JSON.parse((ev as MessageEvent).data) as State)
  })
  es.addEventListener('error', () => onError(true))
  return () => es.close()
}
