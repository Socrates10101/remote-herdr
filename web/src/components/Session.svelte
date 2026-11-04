<script lang="ts" module>
  export type Tab = 'chat' | 'files' | 'previews'
</script>

<script lang="ts">
  import type { Agent } from '../lib/api'
  import { chrome } from '../lib/chrome.svelte'
  import Chat from './Chat.svelte'
  import Files from './Files.svelte'
  import Previews from './Previews.svelte'
  import ApprovalBar from './ApprovalBar.svelte'
  import Composer from './Composer.svelte'

  let {
    agent,
    size,
    tab,
    onallpreviews,
  }: { agent: Agent; size: number; tab: Tab; onallpreviews: () => void } = $props()

  // The approval sheet never hides: it is the one thing that needs an answer.
  const blocked = $derived(agent.status === 'blocked')

  // The preview frame fills the space above the dock, whose height changes
  // with the composer's text and the approval sheet.
  let dockHeight = $state(0)
  $effect(() => {
    document.documentElement.style.setProperty('--dock-h', `${dockHeight}px`)
  })
</script>

{#if tab === 'chat'}
  <Chat {agent} {size} />
{:else if tab === 'previews'}
  <Previews onall={onallpreviews} />
{:else}
  <Files {agent} />
{/if}

<footer class="dock" class:hidden={chrome.dockHidden && !blocked && tab !== 'previews'} bind:clientHeight={dockHeight}>
  {#if blocked}
    <ApprovalBar {agent} />
  {:else}
    <Composer {agent} />
  {/if}
</footer>
