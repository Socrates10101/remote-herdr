<script lang="ts">
  import { api, type Agent, type Preview } from '../lib/api'
  import PreviewRow from './PreviewRow.svelte'

  let { agent, size, onall }: { agent: Agent; size: number; onall: () => void } = $props()

  let entries = $state<Preview[]>([])
  let loaded = $state(false)
  let error = $state('')

  // Refetch when the session changes or its transcript grows (a new URL may
  // have been posted).
  $effect(() => {
    void size
    api
      .agentPreviews(agent)
      .then((r) => {
        entries = r.entries
        error = ''
        loaded = true
      })
      .catch((e) => (error = (e as Error).message))
  })
</script>

<div class="page">
  {#if error}<p class="error">{error}</p>{/if}
  {#if loaded && entries.length === 0}
    <p class="note">このセッションのプレビューはまだありません。</p>
  {:else}
    <ul class="rows">
      {#each entries as p (p.project + '/' + p.name)}
        <PreviewRow {p} />
      {/each}
    </ul>
  {/if}
  <p class="note"><button class="link" onclick={onall}>すべてのプレビュー</button></p>
</div>
