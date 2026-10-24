<script lang="ts">
  import { refKey, type Agent } from '../lib/api'
  import Spark from './Spark.svelte'
  import { shortPath } from '../lib/home.svelte'

  let {
    agents,
    loaded,
    user = '',
    onselect,
    selected = null,
    compact = false,
  }: {
    agents: Agent[]
    loaded: boolean
    user?: string
    onselect: (id: string) => void
    selected?: string | null
    compact?: boolean
  } = $props()

  const groups = $derived(
    [
      { label: '承認が必要', items: agents.filter((a) => a.status === 'blocked') },
      { label: '作業中', items: agents.filter((a) => a.status === 'working') },
      { label: 'そのほか', items: agents.filter((a) => a.status !== 'blocked' && a.status !== 'working') },
    ].filter((g) => g.items.length > 0),
  )

  // Host names only matter once more than one machine is listed.
  const multiHost = $derived(new Set(agents.map((a) => a.host)).size > 1)

  const statusText: Record<string, string> = { idle: '待機中', done: '完了', unknown: '状態不明' }

  function place(cwd: string): string {
    return shortPath(cwd)
  }
</script>

<div class="list-page" class:compact>
  {#if !loaded}
    <p class="empty">herdr に接続しています…</p>
  {:else if agents.length === 0}
    <p class="empty">herdr で動いているエージェントはありません。</p>
  {:else}
    {#each groups as g (g.label)}
      <p class="conv-group">{g.label}</p>
      <ul class="convs">
        {#each g.items as a (refKey(a))}
          <li>
            <button
              class="conv"
              class:active={selected === refKey(a)}
              class:attention={a.status === 'blocked'}
              aria-current={selected === refKey(a) ? 'page' : undefined}
              onclick={() => onselect(refKey(a))}
            >
              <span class="conv-lead">
                {#if a.status === 'working'}
                  <Spark active size={13} />
                {:else}
                  <span class="dot {a.status}"></span>
                {/if}
              </span>
              <span class="conv-main">
                <span class="t">{a.title || a.paneId}</span>
                <span class="conv-meta">{multiHost ? `${a.host} · ` : ''}{place(a.cwd)}</span>
              </span>
              {#if a.status === 'blocked'}
                <span class="wait">承認待ち</span>
              {:else if a.status !== 'working' && !compact}
                <span class="wait quiet">{statusText[a.status] ?? a.status}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/each}
  {/if}

  {#if user && !compact}<p class="footnote">{user} としてログイン中</p>{/if}
</div>
