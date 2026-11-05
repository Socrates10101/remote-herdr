<script lang="ts">
  import { selectedPreview, sessionPreviews } from '../lib/viewer.svelte'
  import PreviewViewer from './PreviewViewer.svelte'

  let { onall }: { onall: () => void } = $props()

  const current = $derived(selectedPreview())
</script>

{#if current}
  <PreviewViewer url={current.url} title={current.name || current.project} version={current.modified} overlay />
{:else if sessionPreviews.loaded}
  <div class="page">
    <p class="note">このセッションのプレビューはまだありません。</p>
    <p class="note"><button class="link" onclick={onall}>すべてのプレビュー</button></p>
  </div>
{/if}
