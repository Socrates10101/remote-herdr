<script lang="ts">
  import type { Preview } from '../lib/api'
  import { ago } from '../lib/time'
  import Icon from './Icon.svelte'

  let { p, showProject = true }: { p: Preview; showProject?: boolean } = $props()

  // Only https links leave the app; the server builds them from -preview-origin.
  const safe = $derived(p.url.startsWith('https://'))
</script>

<li>
  <a class="row preview-row" href={safe ? p.url : undefined} target="_blank" rel="noopener noreferrer">
    <span class="row-lead"><Icon name="preview" size={18} /></span>
    <span class="row-main">
      <span class="row-title">{p.name}</span>
      {#if showProject}<span class="row-sub">{p.project}</span>{/if}
    </span>
    <span class="row-size">{ago(p.modified)}</span>
    <span class="row-trail"><Icon name="external" size={15} /></span>
  </a>
</li>
