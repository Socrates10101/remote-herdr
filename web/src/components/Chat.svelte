<script lang="ts">
  import { api, refKey, type Agent, type AgentRef, type Block, type Message } from '../lib/api'
  import { renderMarkdown } from '../lib/markdown'
  import Icon from './Icon.svelte'
  import Spark from './Spark.svelte'
  import { agentName } from '../lib/agents'

  let { agent, size }: { agent: Agent; size: number } = $props()

  const name = $derived(agentName(agent.agent))

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

  // A run of tool-only entries (calls and their results) collapses into one
  // row so a long stretch of Bash/Read calls does not bury the conversation.
  type Item =
    | { kind: 'msg'; m: Message; idx: number }
    | { kind: 'tools'; key: string; idx: number; blocks: Block[] }

  function isToolEntry(m: Message): boolean {
    return m.role === 'assistant' ? toolsOnly(m) : isToolOnly(m)
  }

  const items = $derived.by(() => {
    const out: Item[] = []
    let run: { idx: number; msgs: Message[] } | null = null
    const flush = () => {
      if (!run) return
      const blocks = run.msgs.flatMap((x) => x.blocks)
      if (blocks.filter((b) => b.kind === 'tool_use').length >= 2) {
        out.push({ kind: 'tools', key: 'g:' + run.msgs[0].uuid, idx: run.idx, blocks })
      } else {
        run.msgs.forEach((m, k) => out.push({ kind: 'msg', m, idx: run!.idx + k }))
      }
      run = null
    }
    messages.forEach((m, idx) => {
      if (isToolEntry(m)) {
        run ??= { idx, msgs: [] }
        run.msgs.push(m)
      } else {
        flush()
        out.push({ kind: 'msg', m, idx })
      }
    })
    flush()
    return out
  })

  function groupLabel(blocks: Block[]): { count: number; names: string; failed: number; last: string } {
    const uses = blocks.filter((b) => b.kind === 'tool_use')
    const byName = new Map<string, number>()
    for (const b of uses) byName.set(b.name ?? '?', (byName.get(b.name ?? '?') ?? 0) + 1)
    const names = [...byName]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 3)
      .map(([n, c]) => (c > 1 ? `${n} ×${c}` : n))
      .join(' · ')
    const last = uses[uses.length - 1]
    return {
      count: uses.length,
      names: byName.size > 3 ? names + ' …' : names,
      failed: blocks.filter((b) => b.kind === 'tool_result' && b.isError).length,
      last: last ? summary(last) : '',
    }
  }

  // An agent entry opens a new turn (and shows the agent's name) when the
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

{#snippet block(b: Block)}
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
{/snippet}

<!-- svelte-ignore a11y_click_events_have_key_events, a11y_no_static_element_interactions -->
<div class="thread" onclick={copyCode}>
  {#if error}<p class="error">{error}</p>{/if}
  {#if !agent.sessionId}
    <p class="empty">このペインの会話記録は見つかりません。</p>
  {/if}

  {#each items as it (it.kind === 'msg' ? it.m.uuid : it.key)}
    {#if it.kind === 'tools'}
      {@const g = groupLabel(it.blocks)}
      <div class="msg-ai compact" class:cont={!opensTurn(it.idx)}>
        <span class="ai-mark">{#if opensTurn(it.idx)}<Spark size={14} />{/if}</span>
        <div>
          {#if opensTurn(it.idx)}
            <div class="msg-head"><b>{name}</b><span>{time(messages[it.idx].timestamp)}</span></div>
          {/if}
          <details class="tool tool-group" class:failed={g.failed > 0}>
            <summary>
              <span class="chev"><Icon name="chevron" size={13} /></span>
              <span class="tool-name">ツール {g.count} 回</span>
              <span class="tool-arg">{g.names}{g.failed ? ` · 失敗 ${g.failed}` : ''}</span>
            </summary>
            <div class="tool-group-body">
              {#each it.blocks as b, i (i)}{@render block(b)}{/each}
            </div>
          </details>
          {#if g.last && agent.status === 'working' && it === items[items.length - 1]}
            <p class="tool-latest">{g.last}</p>
          {/if}
        </div>
      </div>
    {:else}
    {@const m = it.m}
    {@const idx = it.idx}
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
            <div class="msg-head"><b>{name}</b><span>{time(m.timestamp)}</span></div>
          {/if}
          {#each m.blocks as b, i (i)}
            {@render block(b)}
          {/each}
        </div>
      </div>
    {/if}
    {/if}
  {/each}

  {#if agent.status === 'working'}
    <div class="working"><Spark active size={16} /><span>作業中</span></div>
  {:else if agent.status === 'blocked'}
    <div class="working waiting"><Spark size={16} /><span>あなたの答えを待っています</span></div>
  {/if}
</div>
