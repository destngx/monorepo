<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder for a chart: bars rising from a baseline, at the chart's height.
  let { height = 220, label, class: className = '' }: { height?: number; label?: string; class?: string } = $props();
  const bars = [45, 70, 55, 85, 60, 75, 50, 90, 65, 80, 58, 72];
</script>

<Loading {label} class={className}>
  <div class="flex items-end gap-2 border-b border-stone-900/10 dark:border-white/10" style="height: {height}px">
    {#each bars as bar, i (i)}
      <div class="skeleton flex-1 rounded-b-none" style="height: {bar}%"></div>
    {/each}
  </div>
</Loading>
