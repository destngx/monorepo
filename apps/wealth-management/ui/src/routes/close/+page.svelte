<script lang="ts">
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import CloseChecklist from '$lib/close/CloseChecklist.svelte';
  import { closeMonths } from '$lib/close/close';
  import Deferred from '$lib/components/Deferred.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import { currentMonth, formatMonth, shiftMonth } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';

  // ROADMAP R8: the monthly close checklist, ending with the month's snapshot (P3).
  // The header and month links render at once; the checklist streams in, and
  // another month brings its skeleton back (a refresh keeps the shown one).
  let { data } = $props();
  const close = track(() => data.close, { scope: () => data.month });
  // Without ?month= the engine picks the month, known once the checklist answers.
  const shown = $derived(data.month || close.value?.month || '');
  // This month and the two before it, plus the month shown when it is older.
  const months = $derived([...new Set([...(shown ? [shown] : []), ...closeMonths(currentMonth(), shiftMonth)])].sort());

  async function writeSnapshot(month: string) {
    const result = await api.writeSnapshot(month);
    toasts.success(`${result.created ? 'Wrote' : 'Updated'} the ${formatMonth(month)} snapshot.`);
    await invalidateAll();
  }
</script>

<PageHeader title="Monthly close" subtitle="A short checklist to finish the month, ending with its snapshot" />

<nav class="mb-4 flex flex-wrap items-center gap-1" aria-label="Month">
  {#each months as month (month)}
    <a
      href="{base}/close?month={month}"
      class="btn-ghost rounded-md px-2 py-0.5 text-sm font-normal {month === shown
        ? 'bg-indigo-500/15 font-semibold text-indigo-700 hover:bg-indigo-500/20 dark:text-indigo-300'
        : 'muted'}"
      aria-current={month === shown ? 'page' : undefined}>{formatMonth(month)}</a
    >
  {/each}
</nav>

<Deferred data={data.close} scope={data.month} label="the close checklist">
  {#snippet children(value)}
    <div><CloseChecklist close={value} onWriteSnapshot={() => writeSnapshot(value.month)} /></div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonCard lines={2} label="the close checklist" />
      <SkeletonCard title="Checklist" rows={7} lines={0} class="mt-4" />
    </div>
  {/snippet}
</Deferred>

<p class="muted mt-4 text-sm">
  Snapshots build the <a class="link" href="{base}/health#net-worth-history">net worth history</a>: a month with a
  snapshot keeps its price moves, which the estimate from transactions misses.
</p>
