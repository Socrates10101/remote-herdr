<script lang="ts">
  import { api, fileURL, type Agent, type Entry } from '../lib/api'
  import { renderCode, renderMarkdown } from '../lib/markdown'
  import { langForPath } from '../lib/highlight'
  import { dirname, joinPath } from '../lib/paths'
  import Icon from './Icon.svelte'

  let { agent }: { agent: Agent } = $props()

  let dir = $state('')
  let entries = $state<Entry[]>([])
  let open = $state<{ path: string; kind: Entry['kind'] } | null>(null)
  let text = $state('')
  let showSource = $state(false)
  let error = $state('')

  $effect(() => {
    void agent.paneId
    const d = dir
    api
      .files(agent, d)
      .then((r) => {
        entries = r.entries
        error = ''
      })
      .catch((e) => (error = (e as Error).message))
  })

  async function openEntry(e: Entry) {
    const path = joinPath(dir, e.name)
    window.scrollTo({ top: 0 })
    if (e.dir) {
      dir = path
      open = null
      return
    }
    open = { path, kind: e.kind }
    text = ''
    showSource = false
    if (e.kind === 'text') {
      try {
        text = await api.fileText(agent, path)
      } catch (err) {
        error = (err as Error).message
      }
    }
  }

  const crumbs = $derived(dir ? dir.split('/') : [])
  const isMarkdown = $derived(open?.path.toLowerCase().endsWith('.md') ?? false)
  const ctx = $derived({ ref: agent, baseDir: open ? dirname(open.path) : '', cwd: agent.cwd })
  const root = $derived(agent.cwd.split('/').pop() || '/')

  function size(n: number): string {
    if (n < 1024) return `${n} B`
    if (n < 1024 * 1024) return `${Math.round(n / 1024)} KB`
    return `${(n / 1024 / 1024).toFixed(1)} MB`
  }
  const icons: Record<string, string> = { dir: 'folder', image: 'image', pdf: 'pdf', text: 'file', other: 'file' }
</script>

<div class="page">
  {#if error}<p class="error">{error}</p>{/if}

  {#if open}
    <div class="toolbar">
      <button class="icon-btn" onclick={() => (open = null)} aria-label="一覧へ戻る"><Icon name="back" size={20} /></button>
      <span class="file-path">{open.path}</span>
      {#if isMarkdown}
        <div class="mini-seg">
          <button class:active={!showSource} onclick={() => (showSource = false)}>表示</button>
          <button class:active={showSource} onclick={() => (showSource = true)}>ソース</button>
        </div>
      {/if}
    </div>
    <div class="viewer">
      {#if open.kind === 'image'}
        <img class="file-image" alt={open.path} src={fileURL(agent, open.path)} />
      {:else if open.kind === 'pdf'}
        <a class="pill primary" href={fileURL(agent, open.path)} target="_blank" rel="noopener noreferrer">PDF を開く</a>
      {:else if open.kind === 'text'}
        {#if isMarkdown && !showSource}
          <div class="prose">{@html renderMarkdown(text, ctx)}</div>
        {:else}
          <div class="code-view">{@html renderCode(text, langForPath(open.path))}</div>
        {/if}
      {:else}
        <p class="note">このファイルは表示できません。</p>
      {/if}
    </div>
  {:else}
    <nav class="crumbs" aria-label="現在の場所">
      <button onclick={() => (dir = '')}>{root}</button>
      {#each crumbs as c, i (i)}
        <span class="crumb-sep">/</span><button onclick={() => (dir = crumbs.slice(0, i + 1).join('/'))}>{c}</button>
      {/each}
    </nav>
    <ul class="rows files">
      {#if dir}
        <li>
          <button class="row" onclick={() => (dir = dirname(dir))}>
            <span class="row-lead"><Icon name="back" size={18} /></span>
            <span class="row-main"><span class="row-title">ひとつ上へ</span></span>
          </button>
        </li>
      {/if}
      {#each entries as e (e.name)}
        <li>
          <button class="row" onclick={() => openEntry(e)} disabled={e.kind === 'other'}>
            <span class="row-lead file-icon {e.kind}"><Icon name={icons[e.kind]} size={18} /></span>
            <span class="row-main"><span class="row-title">{e.name}</span></span>
            {#if e.dir}
              <span class="row-trail"><Icon name="chevron" size={16} /></span>
            {:else}
              <span class="row-size">{size(e.size)}</span>
            {/if}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
