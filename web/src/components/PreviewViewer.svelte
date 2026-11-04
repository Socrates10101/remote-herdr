<script lang="ts">
  import { closePreview, viewer } from '../lib/viewer.svelte'
  import Icon from './Icon.svelte'

  // The frame runs on the preview origin, never on remote-herdr's (enforced
  // by the server's CSP frame-src). The sandbox lets the page run scripts
  // and open new tabs, but not navigate remote-herdr itself.
  const sandbox = 'allow-scripts allow-same-origin allow-forms allow-popups allow-popups-to-escape-sandbox allow-downloads'

  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape' && viewer.url) closePreview()
  }
</script>

<svelte:window onkeydown={onKey} />

{#if viewer.url}
  <div class="pv" role="dialog" aria-modal="true" aria-label={viewer.title}>
    <div class="pv-bar">
      <button class="icon-btn" onclick={closePreview} aria-label="閉じる"><Icon name="back" size={20} /></button>
      <span class="pv-title" title={viewer.url}>{viewer.title}</span>
      <a class="icon-btn" href={viewer.url} target="_blank" rel="noopener noreferrer" aria-label="新しいタブで開く">
        <Icon name="external" size={17} />
      </a>
    </div>
    {#key viewer.url}
      <iframe class="pv-frame" src={viewer.url} title={viewer.title} {sandbox} referrerpolicy="no-referrer"></iframe>
    {/key}
  </div>
{/if}
