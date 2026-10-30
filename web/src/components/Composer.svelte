<script lang="ts">
  import { api, type Agent } from '../lib/api'
  import Icon from './Icon.svelte'
  import { shellAllowed } from '../lib/home.svelte'

  let { agent }: { agent: Agent } = $props()

  let text = $state('')
  let busy = $state(false)
  let error = $state('')
  let field: HTMLTextAreaElement | undefined = $state()

  // Mirrors the server rule so the reason shows before sending.
  const shellInput = $derived(text.trimStart().startsWith('!'))
  const blockedInput = $derived(shellInput && !shellAllowed())
  const canSend = $derived(!busy && text.trim() !== '' && !blockedInput)

  $effect(() => {
    void text
    if (!field) return
    field.style.height = 'auto'
    field.style.height = Math.min(field.scrollHeight, window.innerHeight * 0.35) + 'px'
  })

  async function send() {
    if (!canSend) return
    busy = true
    error = ''
    try {
      await api.prompt(agent, text)
      text = ''
    } catch (e) {
      error = (e as Error).message
    } finally {
      busy = false
    }
  }

  async function interrupt() {
    busy = true
    error = ''
    try {
      await api.interrupt(agent)
    } catch (e) {
      error = (e as Error).message
    } finally {
      busy = false
    }
  }

  function onkeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault()
      send()
    }
  }
</script>

{#if blockedInput}
  <p class="dock-note">「!」で始まる入力はシェルを直接動かすため、ここからは送れません（`-allow-shell` で許可できます）。</p>
{:else if shellInput}
  <p class="dock-note">「!」で始まる入力は、確認なしでこのマシンのシェルで実行されます。</p>
{:else if error}
  <p class="dock-note error">送れませんでした: {error}</p>
{/if}
<div class="composer">
  <textarea
    bind:this={field}
    bind:value={text}
    {onkeydown}
    rows="1"
    placeholder="Claude に送る"
    aria-label="プロンプト"
  ></textarea>
  <div class="composer-row">
    <span class="hint">⌘ または Ctrl + Enter で送信 · 端末の Claude にそのまま入力されます</span>
    <span class="spacer"></span>
    {#if agent.status === 'working' && !text.trim()}
      <button class="send stop" disabled={busy} onclick={interrupt} aria-label="作業を中断">
        <Icon name="stop" size={15} />
      </button>
    {:else}
      <button class="send" disabled={!canSend} onclick={send} aria-label="送信">
        <Icon name="up" size={17} />
      </button>
    {/if}
  </div>
</div>
