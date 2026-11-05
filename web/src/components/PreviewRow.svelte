<script lang="ts">
  import type { Preview } from '../lib/api'
  import { ago } from '../lib/time'
  import Icon from './Icon.svelte'

  let { p, onopen }: { p: Preview; onopen: (p: Preview) => void } = $props()

  // Only https links are shown; the server builds them from -preview-origin.
  const safe = $derived(p.url.startsWith('https://'))
</script>

<li class="preview-item">
  <button class="row preview-row" disabled={!safe} onclick={() => onopen(p)}>
    <span class="row-lead"><Icon name="preview" size={18} /></span>
    <span class="row-main">
      <span class="row-title">{p.name || p.project}</span>
      {#if p.name}<span class="row-sub">{p.project}</span>{/if}
    </span>
    <span class="row-size">{ago(p.modified)}</span>
  </button>
  {#if safe}
    <a class="icon-btn preview-ext" href={p.url} target="_blank" rel="noopener noreferrer" aria-label="新しいタブで開く">
      <Icon name="external" size={15} />
    </a>
  {/if}
</li>
