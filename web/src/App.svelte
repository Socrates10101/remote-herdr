<script lang="ts">
  import { onMount } from 'svelte'
  import { api, refKey, subscribe, type Agent, type State } from './lib/api'
  import { chrome, showChrome, watchScroll } from './lib/chrome.svelte'
  import AgentList from './components/AgentList.svelte'
  import Session, { type Tab } from './components/Session.svelte'
  import Icon from './components/Icon.svelte'
  import Spark from './components/Spark.svelte'
  import { watchEdgeSwipe } from './lib/swipe'
  import { previewsEnabled, setAllowShell, setHome, setPreviews, shortPath } from './lib/home.svelte'
  import PreviewsPage from './components/PreviewsPage.svelte'
  import PreviewMenu from './components/PreviewMenu.svelte'
  import {
    allPreviews,
    loadAllPreviews,
    loadSessionPreviews,
    resetSessionPreviews,
    selectedPreview,
    sessionPreviews,
  } from './lib/viewer.svelte'
  import type { Preview } from './lib/api'

  let agents = $state<Agent[]>([])
  let sizes = $state<Record<string, number>>({})
  let selected = $state<string | null>(null)
  let tab = $state<Tab>('chat')
  let down = $state(false)
  let loaded = $state(false)
  let user = $state('')

  const current = $derived(agents.find((a) => refKey(a) === selected) ?? null)

  // The all-previews page lives at #previews; session keys always contain "/".
  const PREVIEWS = 'previews'
  const onPreviews = $derived(selected === PREVIEWS && previewsEnabled())

  const tabs = $derived<[Tab, string, string][]>([
    ['chat', '会話', 'chat'],
    ['files', 'ファイル', 'folder'],
    ...(previewsEnabled() ? [['previews', 'プレビュー', 'preview'] as [Tab, string, string]] : []),
  ])

  function show(id: string | null) {
    selected = id
    tab = 'chat'
    resetSessionPreviews(id ?? '')
    showChrome()
    window.scrollTo({ top: 0 })
  }

  // Opening a session pushes a history entry, so the browser's own back
  // (iOS edge swipe, Android back) returns to the list.
  function select(id: string | null) {
    if (id && selected) {
      // Switching between sessions (sidebar) does not stack history.
      history.replaceState({ session: id }, '', `#${encodeURIComponent(id)}`)
      show(id)
    } else if (id) {
      history.pushState({ session: id }, '', `#${encodeURIComponent(id)}`)
      show(id)
    } else if (history.state?.session) {
      history.back() // popstate shows the list
    } else {
      history.replaceState(null, '', '#')
      show(null)
    }
  }

  function fromHash(): string | null {
    return decodeURIComponent(location.hash.slice(1)) || null
  }

  function setTab(t: Tab) {
    tab = t
    sessionPreviews.menu = false
    showChrome()
    window.scrollTo({ top: 0 })
  }

  // The preview tab opens the only preview directly; with several it drops
  // a picker below the tab and the pick is shown in the session area.
  async function previewTab() {
    if (current && !sessionPreviews.loaded) await loadSessionPreviews(current).catch(() => {})
    const list = sessionPreviews.entries
    if (list.length > 1) {
      sessionPreviews.menu = !sessionPreviews.menu
      return
    }
    if (list.length === 1) sessionPreviews.url = list[0].url
    setTab('previews')
  }

  function pickPreview(p: Preview) {
    sessionPreviews.url = p.url
    setTab('previews')
  }

  // From the sidebar: a preview of another open session switches to that
  // session (so the composer talks to the session that made it); one of an
  // ended session is shown in this session's frame.
  function pickAny(p: Preview) {
    if (current && sessionPreviews.entries.some((e) => e.url === p.url)) return pickPreview(p)
    if (openFromAll(p)) return
    sessionPreviews.adhoc = p
    pickPreview(p)
  }

  const previewing = $derived(!!current && tab === 'previews')

  // Sidebar groups while previewing: this session, other open sessions, the rest.
  const sideGroups = $derived.by(() => {
    if (!previewing || !current) return []
    const here = refKey(current)
    const live = new Set(agents.map((a) => refKey(a)))
    const groups = new Map<string, { key: string; label: string; items: Preview[] }>()
    const rest: Preview[] = []
    const mine = sessionPreviews.entries
    for (const p of allPreviews.entries) {
      const o = p.owner
      const key = o?.host && o.paneId ? `${o.host}/${o.paneId}` : ''
      if (key === here || mine.some((m) => m.url === p.url)) continue
      if (!key || !live.has(key)) {
        rest.push(p)
        continue
      }
      const g = groups.get(key) ?? { key, label: o?.title || key, items: [] }
      g.items.push(p)
      groups.set(key, g)
    }
    return [
      ...(mine.length ? [{ key: here, label: 'このセッション', items: mine }] : []),
      ...groups.values(),
      ...(rest.length ? [{ key: '', label: '終わったセッション・持ち主なし', items: rest }] : []),
    ]
  })

  $effect(() => {
    if (!previewing) return
    void (current ? sizes[refKey(current)] : 0)
    loadAllPreviews().catch(() => {})
  })

  // Opening from the all-previews page: jump to the owning session's tab
  // when it is open in herdr; otherwise the page shows it in place.
  function openFromAll(p: Preview): boolean {
    const o = p.owner
    const key = o?.host && o.paneId ? `${o.host}/${o.paneId}` : ''
    if (!key || !agents.some((a) => refKey(a) === key)) return false
    select(key)
    sessionPreviews.url = p.url
    tab = 'previews'
    if (current) loadSessionPreviews(current).catch(() => {})
    return true
  }

  // Keep the session's preview list fresh: on open and whenever its
  // transcript grows (a new preview URL may have been posted).
  $effect(() => {
    const a = current
    void (a ? sizes[refKey(a)] : 0)
    if (!a || !previewsEnabled()) return
    loadSessionPreviews(a).catch(() => {})
  })

  onMount(() => {
    // A deep link still gets a list entry underneath it to go back to.
    const initial = fromHash()
    if (initial) {
      history.replaceState(null, '', '#')
      history.pushState({ session: initial }, '', `#${encodeURIComponent(initial)}`)
      selected = initial
      resetSessionPreviews(initial)
    }
    const onPop = () => show(fromHash())
    window.addEventListener('popstate', onPop)
    const stopSwipe = watchEdgeSwipe(
      () => selected !== null,
      () => select(null),
    )
    api
      .me()
      .then((m) => {
        user = m.user
        setHome(m.home)
        setAllowShell(m.allowShell)
        setPreviews(m.previews)
      })
      .catch(() => {})
    // First paint does not wait for the event stream.
    api
      .agents()
      .then((r) => {
        if (!loaded) {
          agents = r.agents
          loaded = true
        }
      })
      .catch(() => {})
    const stopScroll = watchScroll()
    const stopEvents = subscribe(
      (s: State) => {
        agents = s.agents
        sizes = s.sizes
        loaded = true
      },
      (d) => (down = d),
    )
    return () => {
      stopScroll()
      stopEvents()
      stopSwipe()
      window.removeEventListener('popstate', onPop)
    }
  })
</script>

<aside class="sidebar" aria-label="セッション一覧">
  <div class="brand">
    <span class="brand-mark"><Spark size={16} /></span>
    <b>remote-herdr</b>
    <span class="conn" class:down role="status" aria-label={down ? '切断中' : '接続中'}></span>
  </div>
  {#if previewsEnabled()}
    <div class="side-nav">
      <button class="side-link" class:active={onPreviews} onclick={() => select(PREVIEWS)}>
        <Icon name="preview" size={16} />プレビュー
      </button>
    </div>
  {/if}
  <div class="conv-list">
    {#each sideGroups as g (g.key + g.label)}
      {#snippet rows(items: Preview[])}
        <ul class="convs">
          {#each items as p (p.url)}
            <li>
              <button
                class="conv side-preview"
                class:active={p.url === sessionPreviews.url}
                aria-current={p.url === sessionPreviews.url ? 'page' : undefined}
                title={p.project + '/' + p.name}
                onclick={() => pickAny(p)}
              >
                <span class="conv-lead"><Icon name="preview" size={14} /></span>
                <span class="conv-main"><span class="t">{p.name || p.project}</span></span>
              </button>
            </li>
          {/each}
        </ul>
      {/snippet}
      {#if g.key === ''}
        <details class="side-rest">
          <summary class="conv-group">{g.label}（{g.items.length}）</summary>
          {@render rows(g.items)}
        </details>
      {:else}
        <p class="conv-group">{g.label}</p>
        {@render rows(g.items)}
      {/if}
    {/each}
    {#if sideGroups.length > 0}<p class="conv-group side-sep">セッション</p>{/if}
    <AgentList {agents} {loaded} onselect={select} {selected} compact />
  </div>
  {#if user}
    <div class="side-foot">
      <span class="avatar">{user.slice(0, 2).toUpperCase()}</span>
      <span class="side-user">{user}</span>
    </div>
  {/if}
</aside>

<header
  class="topbar"
  class:over={!!current && tab === 'previews' && !!selectedPreview()}
  class:hidden={chrome.hidden && !!current}
  class:list-only={!current && !onPreviews}
>
  {#if current}
    <button class="icon-btn back-button" onclick={() => select(null)} aria-label="セッション一覧へ戻る">
      <Icon name="back" size={20} />
    </button>
    <h1 title={current.cwd}>{current.title || current.paneId}</h1>
    <span class="topbar-sub">{shortPath(current.cwd)}</span>
    <span class="spacer"></span>
    <div class="tabs" role="tablist" aria-label="表示の切り替え">
      {#each tabs as [id, label, icon] (id)}
        {#if id === 'previews'}
          <button
            role="tab"
            class="pmenu-anchor"
            aria-selected={tab === id}
            aria-haspopup={sessionPreviews.entries.length > 1 ? 'menu' : undefined}
            aria-expanded={sessionPreviews.entries.length > 1 ? sessionPreviews.menu : undefined}
            class:on={tab === id}
            onclick={previewTab}
          >
            <Icon name={icon} size={15} />{label}
            {#if sessionPreviews.entries.length > 0}<span class="tab-count">{sessionPreviews.entries.length}</span>{/if}
          </button>
        {:else}
          <button role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => setTab(id)}>
            <Icon name={icon} size={15} />{label}
          </button>
        {/if}
      {/each}
      {#if tab === 'previews' && selectedPreview()}
        <a
          class="icon-btn tab-ext"
          href={selectedPreview()?.url}
          target="_blank"
          rel="noopener noreferrer"
          aria-label="新しいタブで開く"
          title={selectedPreview()?.name || selectedPreview()?.project}
        >
          <Icon name="external" size={16} />
        </a>
      {/if}
      {#if sessionPreviews.menu}
        <PreviewMenu
          entries={sessionPreviews.entries}
          current={tab === 'previews' ? sessionPreviews.url : ''}
          onpick={pickPreview}
          onall={() => select(PREVIEWS)}
          onclose={() => (sessionPreviews.menu = false)}
        />
      {/if}
    </div>
  {:else if onPreviews}
    <button class="icon-btn back-button" onclick={() => select(null)} aria-label="セッション一覧へ戻る">
      <Icon name="back" size={20} />
    </button>
    <h1>プレビュー</h1>
    <span class="spacer"></span>
  {:else}
    <span class="brand-mark"><Spark size={16} /></span>
    <h1>remote-herdr</h1>
    <span class="spacer"></span>
    <span class="conn" class:down role="status" aria-label={down ? '切断中' : '接続中'}></span>
  {/if}
</header>

<main class:with-session={!!current}>
  {#if current}
    <Session agent={current} size={sizes[refKey(current)] ?? -1} {tab} onallpreviews={() => select(PREVIEWS)} />
  {:else if onPreviews}
    <PreviewsPage {agents} onselect={select} onopen={openFromAll} />
  {:else if selected && loaded && selected !== PREVIEWS}
    <p class="empty">このペインは閉じられたようです。<button class="link" onclick={() => select(null)}>一覧へ戻る</button></p>
  {:else}
    <div class="narrow-only list-page-wrap">
      {#if previewsEnabled()}
        <button class="side-link list-previews" onclick={() => select(PREVIEWS)}>
          <Icon name="preview" size={16} />プレビュー<span class="spacer"></span><Icon name="chevron" size={14} />
        </button>
      {/if}
      <AgentList {agents} {loaded} {user} onselect={select} />
    </div>
    <p class="wide-only pick-hint">左の一覧からセッションを選んでください。</p>
  {/if}
</main>

