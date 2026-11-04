<script lang="ts">
  import Icon from './Icon.svelte'

  let {
    url,
    title,
    withDock = true,
    onback,
  }: { url: string; title: string; withDock?: boolean; onback?: () => void } = $props()

  // The frame runs on the preview origin, never on remote-herdr's (enforced
  // by the server's CSP frame-src). The sandbox lets the page run scripts
  // and open new tabs, but not navigate remote-herdr itself.
  const sandbox =
    'allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads'
</script>

<div class="pframe" class:with-dock={withDock}>
  <div class="pframe-bar">
    {#if onback}
      <button class="icon-btn" onclick={onback} aria-label="一覧へ戻る"><Icon name="back" size={18} /></button>
    {/if}
    <span class="pframe-title" title={url}>{title}</span>
    <a class="icon-btn" href={url} target="_blank" rel="noopener noreferrer" aria-label="新しいタブで開く">
      <Icon name="external" size={16} />
    </a>
  </div>
  {#key url}
    <iframe src={url} {title} {sandbox} referrerpolicy="no-referrer"></iframe>
  {/key}
</div>
