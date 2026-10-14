<script lang="ts">
  import type { DataSource } from '$lib/api/types';
  import { sourceLabel, sourceState, sourceTitle, type SourceState } from '$lib/source';
  import Icon from './Icon.svelte';

  // Where a market figure came from (WM-INV-15). Muted when the first provider
  // answered; amber after a fallback; orange when every provider failed and
  // the figure is the last good copy. `detail` qualifies it, e.g. a rate basis.
  let { source, detail }: { source?: DataSource; detail?: string } = $props();

  const tones: Record<SourceState, string> = {
    live: 'bg-slate-900/5 text-slate-600 dark:bg-white/10 dark:text-slate-300',
    fallback: 'bg-amber-500/15 text-amber-700 dark:text-amber-300',
    stale: 'bg-orange-500/15 text-orange-700 dark:text-orange-300',
  };
  const notes: Record<SourceState, string> = { live: '', fallback: 'Fallback source:', stale: 'Stale data:' };
  const state = $derived(source ? sourceState(source) : null);
</script>

{#if source && state}
  <span
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium whitespace-nowrap {tones[state]}"
    title={sourceTitle(source, detail)}
  >
    {#if state === 'fallback'}<Icon name="alert" size={12} />{:else if state === 'stale'}<Icon
        name="clock"
        size={12}
      />{/if}
    {#if notes[state]}<span class="sr-only">{notes[state]}</span>{/if}
    {sourceLabel(source, detail)}
  </span>
{:else}
  <span class="muted text-xs">-</span>
{/if}
