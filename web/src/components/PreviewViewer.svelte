<script lang="ts">
  import Icon from './Icon.svelte'
  import { followFrameScroll, resetFrameScroll } from '../lib/chrome.svelte'

  let {
    url,
    title,
    overlay = false,
    onback,
  }: { url: string; title: string; overlay?: boolean; onback?: () => void } = $props()

  // The frame runs on the preview origin, never on remote-herdr's (enforced
  // by the server's CSP frame-src). The sandbox lets the page run scripts
  // and open new tabs, but not navigate remote-herdr itself.
  let frame = $state<HTMLIFrameElement | null>(null)

  // A preview server may add a script that reports the page's scroll as
  // {type: 'remote-herdr:scroll', y, max}. Only messages from this frame's
  // window and origin count, and they only show or hide the chrome.
  function onMessage(e: MessageEvent) {
    if (!overlay || !frame || e.source !== frame.contentWindow) return
    let origin = ''
    try {
      origin = new URL(url).origin
    } catch {
      return
    }
    const d = e.data as { type?: unknown; y?: unknown; max?: unknown } | null
    if (e.origin !== origin || !d || d.type !== 'remote-herdr:scroll') return
    if (typeof d.y !== 'number' || !Number.isFinite(d.y)) return
    followFrameScroll(d.y, typeof d.max === 'number' && Number.isFinite(d.max) ? d.max : 0)
  }

  $effect(() => {
    void url
    resetFrameScroll()
  })

  const sandbox =
    'allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads'
</script>

<svelte:window onmessage={onMessage} />

<!-- overlay: fills the session area; the top bar and the composer float over
     it translucently (the page's controls live in the top bar). -->
<div class="pframe" class:overlay>
  {#if !overlay}
  <div class="pframe-bar">
    {#if onback}
      <button class="icon-btn" onclick={onback} aria-label="一覧へ戻る"><Icon name="back" size={18} /></button>
    {/if}
    <span class="pframe-title" title={url}>{title}</span>
    <a class="icon-btn" href={url} target="_blank" rel="noopener noreferrer" aria-label="新しいタブで開く">
      <Icon name="external" size={16} />
    </a>
  </div>
  {/if}
  {#key url}
    <iframe bind:this={frame} src={url} {title} {sandbox} referrerpolicy="no-referrer"></iframe>
  {/key}
</div>
