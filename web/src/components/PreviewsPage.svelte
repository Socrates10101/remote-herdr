<script lang="ts">
  import { onMount } from 'svelte'
  import { api, refKey, type Agent, type Preview, type PreviewApp } from '../lib/api'
  import PreviewRow from './PreviewRow.svelte'
  import PreviewViewer from './PreviewViewer.svelte'

  let {
    agents,
    onselect,
    onopen,
  }: { agents: Agent[]; onselect: (id: string) => void; onopen: (p: Preview) => boolean } = $props()

  // A preview whose session is not open in herdr is shown here in place.
  let open = $state<{ url: string; title: string } | null>(null)

  function openRow(p: Preview) {
    if (!onopen(p)) open = { url: p.url, title: p.name || p.project }
  }

  let entries = $state<Preview[]>([])
  let apps = $state<PreviewApp[]>([])
  let loaded = $state(false)
  let error = $state('')

  function load() {
    api
      .previews()
      .then((r) => {
        entries = r.entries
        apps = r.apps
        error = ''
        loaded = true
      })
      .catch((e) => (error = (e as Error).message))
  }
  onMount(load)

  // Folders of sessions open in herdr, grouped by session (newest first);
  // the rest (ended sessions, no owner) folded below.
  const groups = $derived.by(() => {
    const live = new Set(agents.map((a) => refKey(a)))
    const map = new Map<string, { key: string; title: string; items: Preview[] }>()
    const rest: Preview[] = []
    for (const p of entries) {
      const o = p.owner
      const key = o?.host && o.paneId ? `${o.host}/${o.paneId}` : ''
      if (!key || !live.has(key)) {
        rest.push(p)
        continue
      }
      const g = map.get(key) ?? { key, title: o?.title || o?.paneId || key, items: [] }
      g.items.push(p)
      map.set(key, g)
    }
    return { live: [...map.values()], rest }
  })
</script>

{#if open}
  <PreviewViewer url={open.url} title={open.title} onback={() => (open = null)} />
{:else}
<div class="page">
  {#if error}<p class="error">{error}</p>{/if}

  {#if apps.length > 0}
    <p class="conv-group">動く画面</p>
    <ul class="rows">
      {#each apps as a (a.port)}
        <li>
          {#if a.listening}
            <button class="row preview-row" onclick={() => (open = { url: a.url, title: `:${a.port}` })}>
              <span class="row-main"><span class="row-title">:{a.port}</span></span>
              <span class="row-size">使用中</span>
            </button>
          {:else}
            <span class="row preview-row idle">
              <span class="row-main"><span class="row-title">:{a.port}</span></span>
              <span class="row-size">空き</span>
            </span>
          {/if}
        </li>
      {/each}
    </ul>
  {/if}

  {#if loaded && entries.length === 0}
    <p class="note">プレビューはまだありません。</p>
  {/if}

  {#each groups.live as g (g.key)}
    <p class="conv-group"><button class="link quiet" onclick={() => onselect(g.key)}>{g.title}</button></p>
    <ul class="rows">
      {#each g.items as p (p.project + '/' + p.name)}
        <PreviewRow {p} onopen={openRow} />
      {/each}
    </ul>
  {/each}

  {#if groups.rest.length > 0}
    <details class="preview-rest">
      <summary class="conv-group">終わったセッション・持ち主なし（{groups.rest.length}）</summary>
      <ul class="rows">
        {#each groups.rest as p (p.project + '/' + p.name)}
          <PreviewRow {p} onopen={openRow} />
        {/each}
      </ul>
    </details>
  {/if}
</div>
{/if}
