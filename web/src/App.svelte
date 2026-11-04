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
  import PreviewViewer from './components/PreviewViewer.svelte'
  import { closePreview, onPopPreview, viewer } from './lib/viewer.svelte'

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
    showChrome()
    window.scrollTo({ top: 0 })
  }

  onMount(() => {
    // A deep link still gets a list entry underneath it to go back to.
    const initial = fromHash()
    if (initial) {
      history.replaceState(null, '', '#')
      history.pushState({ session: initial }, '', `#${encodeURIComponent(initial)}`)
      selected = initial
    }
    const onPop = () => {
      if (onPopPreview()) return
      show(fromHash())
    }
    window.addEventListener('popstate', onPop)
    const stopSwipe = watchEdgeSwipe(
      () => selected !== null || !!viewer.url,
      () => (viewer.url ? closePreview() : select(null)),
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
    <AgentList {agents} {loaded} onselect={select} {selected} compact />
  </div>
  {#if user}
    <div class="side-foot">
      <span class="avatar">{user.slice(0, 2).toUpperCase()}</span>
      <span class="side-user">{user}</span>
    </div>
  {/if}
</aside>

<header class="topbar" class:hidden={chrome.hidden && !!current} class:list-only={!current && !onPreviews}>
  {#if current}
    <button class="icon-btn back-button" onclick={() => select(null)} aria-label="セッション一覧へ戻る">
      <Icon name="back" size={20} />
    </button>
    <h1 title={current.cwd}>{current.title || current.paneId}</h1>
    <span class="topbar-sub">{shortPath(current.cwd)}</span>
    <span class="spacer"></span>
    <div class="tabs" role="tablist" aria-label="表示の切り替え">
      {#each tabs as [id, label, icon] (id)}
        <button role="tab" aria-selected={tab === id} class:on={tab === id} onclick={() => setTab(id)}>
          <Icon name={icon} size={15} />{label}
        </button>
      {/each}
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
    <PreviewsPage {agents} onselect={select} />
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

<PreviewViewer />
