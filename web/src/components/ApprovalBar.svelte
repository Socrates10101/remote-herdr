<script lang="ts">
  import { api, type Agent, type Dialog } from '../lib/api'
  import Icon from './Icon.svelte'
  import { agentName } from '../lib/agents'

  let { agent }: { agent: Agent } = $props()

  const name = $derived(agentName(agent.agent))

  let dialog = $state<Dialog | null>(null)
  let busy = $state(false)
  let error = $state('')
  let loaded = $state(false)

  async function load() {
    try {
      dialog = (await api.dialog(agent)).dialog
      loaded = true
    } catch (e) {
      error = (e as Error).message
    }
  }

  // Multi-question dialogs stay "blocked" while the page changes, so poll.
  $effect(() => {
    void agent.paneId
    load()
    const t = setInterval(load, 1500)
    return () => clearInterval(t)
  })

  const messages: Record<string, string> = {
    'dialog changed; reload': '画面が変わりました。もう一度選んでください。',
    'free-text answers must be typed in the terminal': '自由入力は端末で答えてください。',
  }

  async function choose(index: number, label: string) {
    busy = true
    error = ''
    try {
      await api.choose(agent, index, label)
    } catch (e) {
      const m = (e as Error).message
      error = messages[m] ?? m
    } finally {
      busy = false
      setTimeout(load, 350)
    }
  }

  async function cancel() {
    busy = true
    error = ''
    try {
      await api.decision(agent, 'deny')
    } catch (e) {
      error = (e as Error).message
    } finally {
      busy = false
    }
  }

  const multi = $derived(dialog?.items.some((i) => i.checked !== undefined) ?? false)
</script>

<div class="approval" role="alertdialog" aria-labelledby="approval-title">
  <p id="approval-title" class="approval-title">{name} が答えを待っています</p>
  {#if dialog && dialog.context.length > 0}
    <pre class="approval-context">{dialog.context.map((l) => l.replace(/^←\s*/, '').replace(/\s*→$/, '')).join('\n')}</pre>
  {/if}
  {#if error}<p class="dock-note error">{error}</p>{/if}

  {#if dialog && dialog.items.length > 0}
    <ul class="choices" class:multi>
      {#each dialog.items as item, i (i + item.label)}
        <li>
          <button
            class="choice"
            class:submit={item.label === 'Submit'}
            class:checked={item.checked}
            disabled={busy || item.freeText}
            onclick={() => choose(i, item.label)}
          >
            {#if item.checked !== undefined}
              <span class="check" aria-hidden="true">{#if item.checked}<Icon name="result" size={14} />{/if}</span>
            {/if}
            <span class="choice-main">
              <span class="choice-label">{item.label}</span>
              {#if item.freeText}
                <span class="choice-desc">自由入力は端末で</span>
              {:else if item.description}
                <span class="choice-desc">{item.description}</span>
              {/if}
            </span>
          </button>
        </li>
      {/each}
    </ul>
    {#if multi}<p class="dock-note">項目をタップして選び、最後に Submit を押します。</p>{/if}
  {:else if loaded}
    <p class="dock-note">選択肢を読み取れませんでした。端末で答えてください。</p>
  {/if}

  <div class="approval-actions">
    <button class="pill secondary" disabled={busy} onclick={cancel}>キャンセル（Esc）</button>
  </div>
</div>
