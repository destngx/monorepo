<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder text lines; the last one is shorter, like a paragraph end.
  let { lines = 3, label, class: className = '' }: { lines?: number; label?: string; class?: string } = $props();
</script>

<Loading {label} class={className}>
  <div class="space-y-2">
    {#each { length: lines }, i (i)}
      <div class="skeleton h-3.5 {i === lines - 1 && lines > 1 ? 'w-3/5' : 'w-full'}"></div>
    {/each}
  </div>
</Loading>
