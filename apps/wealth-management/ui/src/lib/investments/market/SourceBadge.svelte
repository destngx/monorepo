<script lang="ts">
  import Icon from '$lib/components/Icon.svelte';

  // Which data source served a series; a fallback lists the providers that failed first.
  let { provider, fallback }: { provider?: string; fallback?: string } = $props();
</script>

{#if provider}
  <span
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap {fallback
      ? 'bg-amber-500/15 text-amber-700 dark:text-amber-300'
      : 'bg-slate-900/5 text-slate-700 dark:bg-white/10 dark:text-slate-300'}"
    title={fallback ? `Served by ${provider} after fallback: ${fallback}` : `Served by ${provider}`}
  >
    {#if fallback}<Icon name="alert" size={12} /><span class="sr-only">Fallback source:</span>{/if}
    {provider}
  </span>
{:else}
  <span class="muted text-xs">-</span>
{/if}
