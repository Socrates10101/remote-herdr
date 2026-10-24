<script lang="ts">
  import { api, refKey, type Agent, type AgentRef, type Block, type Message } from '../lib/api'
  import { renderMarkdown } from '../lib/markdown'
  import Icon from './Icon.svelte'
  import Spark from './Spark.svelte'

  let { agent, size }: { agent: Agent; size: number } = $props()

  let messages = $state<Message[]>([])
  let error = $state('')
  let first = true
  let loadedSize = -2
  let loadedPane = ''

  function nearBottom(): boolean {
    const el = document.documentElement
    return el.scrollHeight - window.scrollY - window.innerHeight < 160
  }

  async function load(ref: AgentRef) {
    const stick = first || nearBottom()
    try {
      const res = await api.transcript(ref)
      if (refKey(ref) !== refKey(agent)) return
      messages = res.messages
      error = ''
      if (stick) {
        requestAnimationFrame(() => window.scrollTo({ top: document.documentElement.scrollHeight }))
        first = false
      }
    } catch (e) {
      error = (e as Error).message
    }
  }

  $effect(() => {
    const key = refKey(agent)
    const s = size
    if (key !== loadedPane) first = true
    if (key !== loadedPane || s !== loadedSize) {
      loadedPane = key
      loadedSize = s
      load({ host: agent.host, paneId: agent.paneId })
    }
  })

  const ctx = $derived({ ref: agent, baseDir: '', cwd: agent.cwd })

  // A user entry that only carries tool results belongs to the agent's turn.
  function isToolOnly(m: Message): boolean {
    return m.role === 'user' && m.blocks.every((b) => b.kind === 'tool_result' || b.kind === 'image')
  }

  function toolsOnly(m: Message): boolean {
    return m.blocks.every((b) => b.kind === 'tool_use' || b.kind === 'tool_result')
  }

  function summary(b: Block): string {
    if (!b.input) return ''
    try {
      const v = JSON.parse(b.input) as Record<string, unknown>
      for (const k of ['command', 'file_path', 'path', 'pattern', 'url', 'description', 'query', 'prompt']) {
        if (typeof v[k] === 'string' && v[k]) return (v[k] as string).split('\n')[0].slice(0, 90)
      }
    } catch {
      /* not JSON */
    }
    return ''
  }

  // An agent entry opens a new turn (and shows the Claude header) when the
  // previous entry was the owner's own message.
  function opensTurn(i: number): boolean {
    if (i === 0) return true
    const prev = messages[i - 1]
    return prev.role === 'user' && !isToolOnly(prev)
  }

  async function copyCode(e: MouseEvent) {
    const btn = (e.target as Element | null)?.closest('.copy')
    if (!btn) return
    const code = btn.closest('.code')?.querySelector('pre')?.textContent ?? ''
    try {
      await navigator.clipboard.writeText(code)
      btn.textContent = 'コピーしました'
      setTimeout(() => (btn.textContent = 'コピー'), 1500)
    } catch {
      btn.textContent = 'コピーできません'
    }
  }

  function time(ts: string): string {
    const d = new Date(ts)
    return isNaN(d.getTime()) ? '' : d.toLocaleTimeString('ja-JP', { hour: '2-digit', minute: '2-digit' })
  }

  function firstLine(s = ''): string {
    const line = s.trim().split('\n')[0] ?? ''
    return line.length > 90 ? line.slice(0, 90) + '…' : line
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="thread" onclick={copyCode}>
  {#if error}<p class="error">{error}</p>{/if}
  {#if !agent.sessionId}
    <p class="empty">このペインの会話記録は見つかりません。</p>
  {/if}

  {#each messages as m, idx (m.uuid)}
    {#if m.role === 'user' && !isToolOnly(m)}
      <div class="msg-user-wrap">
        {#each m.blocks as b, i (i)}
          {#if b.kind === 'text'}
            <div class="msg-user">{b.text?.trim()}</div>
          {:else if b.kind === 'image' && b.data}
            <img class="attachment" alt="添付画像" src={`data:${b.mediaType};base64,${b.data}`} />
          {/if}
        {/each}
      </div>
    {:else}
      <div class="msg-ai" class:cont={!opensTurn(idx)} class:compact={toolsOnly(m)}>
        <span class="ai-mark">{#if opensTurn(idx)}<Spark size={14} />{/if}</span>
        <div>
          {#if opensTurn(idx)}
            <div class="msg-head"><b>Claude</b><span>{time(m.timestamp)}</span></div>
          {/if}
          {#each m.blocks as b, i (i)}
            {#if b.kind === 'text'}
              <div class="md">{@html renderMarkdown(b.text ?? '', ctx)}</div>
            {:else if b.kind === 'tool_use'}
              <details class="tool">
                <summary>
                  <span class="chev"><Icon name="chevron" size={13} /></span>
                  <span class="tool-name">{b.name}</span>
                  <span class="tool-arg">{summary(b)}</span>
                </summary>
                <pre class="tool-body">{b.input}</pre>
              </details>
            {:else if b.kind === 'tool_result' && b.text}
              <details class="tool" class:failed={b.isError}>
                <summary>
                  <span class="chev"><Icon name="chevron" size={13} /></span>
                  <span class="tool-arg">{firstLine(b.text) || (b.isError ? 'エラー' : '結果')}</span>
                </summary>
                <pre class="tool-body">{b.text}{b.truncated ? '\n…（以下省略）' : ''}</pre>
              </details>
            {:else if b.kind === 'image'}
              {#if b.data}
                <img class="attachment" alt="ツールが読み込んだ画像" src={`data:${b.mediaType};base64,${b.data}`} />
              {:else}
                <p class="note">画像が大きいため省略しました。</p>
              {/if}
            {/if}
          {/each}
        </div>
      </div>
    {/if}
  {/each}

  {#if agent.status === 'working'}
    <div class="working"><Spark active size={16} /><span>作業中</span></div>
  {:else if agent.status === 'blocked'}
    <div class="working waiting"><Spark size={16} /><span>あなたの答えを待っています</span></div>
  {/if}
</div>
