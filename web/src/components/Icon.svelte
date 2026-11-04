<script lang="ts" module>
  // One stroke family (24px grid, 1.75 stroke, round caps) for every icon.
  const paths: Record<string, string> = {
    back: 'M15 5l-7 7 7 7',
    chevron: 'M9 5l7 7-7 7',
    send: 'M12 19V5M5.5 11.5L12 5l6.5 6.5',
    stop: 'M7 7h10v10H7z',
    folder: 'M3.5 7.5a2 2 0 012-2h4l2 2h7a2 2 0 012 2v8a2 2 0 01-2 2h-13a2 2 0 01-2-2z',
    file: 'M7 3.5h6.5L18 8v12.5H7zM13.5 3.5V8H18',
    image: 'M4.5 5.5h15v13h-15zM4.5 15l4-4 4 4 3-3 4 4M15 9.25h.01',
    pdf: 'M7 3.5h6.5L18 8v12.5H7zM13.5 3.5V8H18M9.5 13h5M9.5 16.5h3.5',
    tool: 'M14.5 6.5a4 4 0 00-5.2 5.2L4.5 16.5v3h3l4.8-4.8a4 4 0 005.2-5.2l-2.3 2.3-2.4-.6-.6-2.4z',
    result: 'M5 12.5l4.5 4.5L19 7.5',
    error: 'M12 7.5v6M12 16.5h.01M10.3 4.3L3 17a2 2 0 001.7 3h14.6a2 2 0 001.7-3L13.7 4.3a2 2 0 00-3.4 0z',
    terminal: 'M4.5 5.5h15v13h-15zM8 10l2.5 2L8 14M12.5 14H16',
    chat: 'M5 6.5h14v9H10l-4 3v-3H5z',
    refresh: 'M19 12a7 7 0 11-2.05-4.95M19 5v4h-4',
    up: 'M12 19V5M6 11l6-6 6 6',
    preview: 'M4.5 5.5h15v13h-15zM4.5 9h15M7 7.25h.01M9 7.25h.01',
    external: 'M13.5 4.5h6v6M19.5 4.5L11 13M17.5 14v4.5a1 1 0 01-1 1h-11a1 1 0 01-1-1v-11a1 1 0 011-1H10',
  }
</script>

<script lang="ts">
  let { name, size = 20, label = '' }: { name: keyof typeof paths | string; size?: number; label?: string } = $props()
</script>

<svg
  class="icon"
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width="1.75"
  stroke-linecap="round"
  stroke-linejoin="round"
  role={label ? 'img' : undefined}
  aria-label={label || undefined}
  aria-hidden={label ? undefined : 'true'}
>
  <path d={paths[name] ?? ''} />
</svg>
