<script lang="ts">
  import type { Preview } from '../lib/api'
  import { ago } from '../lib/time'

  let {
    entries,
    current,
    onpick,
    onall,
    onclose,
  }: {
    entries: Preview[]
    current: string
    onpick: (p: Preview) => void
    onall: () => void
    onclose: () => void
  } = $props()

  function onWindowClick(e: MouseEvent) {
    const t = e.target as Element | null
    if (!t?.closest('.pmenu, .pmenu-anchor')) onclose()
  }
  function onKey(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose()
  }
</script>

<svelte:window onclick={onWindowClick} onkeydown={onKey} />

<div class="pmenu" role="menu" aria-label="プレビューを選ぶ">
  {#each entries as p (p.url)}
    <button role="menuitem" class="pmenu-item" class:on={p.url === current} onclick={() => onpick(p)}>
      <span class="pmenu-main">
        <span class="pmenu-name">{p.name || p.project}</span>
        {#if p.name}<span class="pmenu-sub">{p.project}</span>{/if}
      </span>
      <span class="pmenu-ago">{ago(p.modified)}</span>
    </button>
  {/each}
  <button role="menuitem" class="pmenu-all" onclick={onall}>すべてのプレビュー</button>
</div>
