<script lang="ts">
  // The working indicator: an eight-ray spark that turns while the agent works.
  let { active = false, size = 18 }: { active?: boolean; size?: number } = $props()
  const rays = Array.from({ length: 8 }, (_, i) => i * 45)
</script>

<svg class="spark" class:active width={size} height={size} viewBox="0 0 24 24" aria-hidden="true">
  {#each rays as deg (deg)}
    <line x1="12" y1="3" x2="12" y2="9" transform={`rotate(${deg} 12 12)`} />
  {/each}
</svg>
