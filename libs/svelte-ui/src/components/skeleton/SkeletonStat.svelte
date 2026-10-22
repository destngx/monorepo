<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder for a Stat tile: label, value and hint, same panel tile.
  let { label, class: className = '' }: { label?: string; class?: string } = $props();
</script>

<Loading {label} class="panel p-4 {className}">
  <div class="skeleton h-3 w-1/3"></div>
  <div class="skeleton mt-3 h-6 w-2/3"></div>
  <div class="skeleton mt-2 h-3 w-1/2"></div>
</Loading>
