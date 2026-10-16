<script lang="ts">
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import CloseChecklist from '$lib/close/CloseChecklist.svelte';
  import { closeMonths } from '$lib/close/close';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { currentMonth, formatMonth, shiftMonth } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';

  // ROADMAP R8: the monthly close checklist, ending with the month's snapshot (P3).
  let { data } = $props();
  const close = $derived(data.close);
  // This month and the two before it, plus the month shown when it is older.
  const months = $derived([...new Set([close.month, ...closeMonths(currentMonth(), shiftMonth)])].sort());
  let writing = $state(false);

  async function writeSnapshot() {
    writing = true;
    try {
      const result = await api.writeSnapshot(close.month);
      toasts.info(`${result.created ? 'Wrote' : 'Updated'} the ${formatMonth(close.month)} snapshot.`);
    } catch (error) {
      toasts.error(error, 'Could not write the snapshot');
    } finally {
      await invalidateAll();
      writing = false;
    }
  }
</script>

<PageHeader title="Monthly close" subtitle="A short checklist to finish the month, ending with its snapshot" />

<nav class="mb-4 flex flex-wrap items-center gap-1" aria-label="Month">
  {#each months as month (month)}
    <a
      href="{base}/close?month={month}"
      class="rounded-md px-2 py-0.5 text-sm {month === close.month
        ? 'bg-indigo-500/15 font-semibold text-indigo-700 dark:text-indigo-300'
        : 'muted'}"
      aria-current={month === close.month ? 'page' : undefined}>{formatMonth(month)}</a
    >
  {/each}
</nav>

<CloseChecklist {close} onWriteSnapshot={writeSnapshot} {writing} />

<p class="muted mt-4 text-sm">
  Snapshots build the <a
    class="text-indigo-600 hover:underline dark:text-indigo-400"
    href="{base}/health#net-worth-history">net worth history</a
  >: a month with a snapshot keeps its price moves, which the estimate from transactions misses.
</p>
