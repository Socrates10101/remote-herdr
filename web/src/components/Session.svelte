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
</script>

{#if tab === 'chat'}
  <Chat {agent} {size} />
{:else if tab === 'previews'}
  <Previews onall={onallpreviews} />
{:else}
  <Files {agent} />
{/if}

<footer class="dock" class:over={tab === 'previews'} class:hidden={chrome.dockHidden && !blocked && tab !== 'previews'}>
  {#if blocked}
    <ApprovalBar {agent} />
  {:else}
    <Composer {agent} />
  {/if}
</footer>
