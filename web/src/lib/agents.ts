// agentName is how the UI names the agent running in a pane. herdr reports
// its kind ("claude", "codex", ...); unknown kinds are shown capitalised.
export function agentName(kind: string): string {
  if (kind === 'codex') return 'Codex'
  if (kind === 'claude' || kind === '') return 'Claude'
  return kind.charAt(0).toUpperCase() + kind.slice(1)
}
