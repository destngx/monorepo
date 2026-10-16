<script lang="ts">
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { BudgetHistory } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatMonth, formatPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { wallRows, type WallCell } from './history';
  import { statusLabel, statusTone } from './status';

  // "The Wall" (WM-BUD-7): six months of spent against limit per category.
  let { month }: { month: string } = $props();

  const MONTHS = 6;
  let history = $state<BudgetHistory | null>(null);
  let error = $state('');
  let loading = $state(false);

  async function load(selected: string) {
    loading = true;
    error = '';
    try {
      history = await api.budgetHistory(selected, MONTHS);
    } catch (failure) {
      history = null;
      error =
        failure instanceof ApiError && failure.status === 404 && !failure.code
          ? 'This engine does not provide budget history yet.'
          : failure instanceof Error
            ? failure.message
            : String(failure);
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    void load(month);
  });

  const rows = $derived(history ? wallRows(history) : []);

  const wash: Record<WallCell['status'], string> = {
    healthy: STATUS.good.wash,
    warning: STATUS.warning.wash,
    urgent: STATUS.critical.wash,
    unbudgeted: 'bg-violet-500/15',
    empty: '',
  };
  // Over-limit cells carry the status icon too, so the wall does not read by colour alone.
  const flagged = (status: WallCell['status']) => (status === 'warning' || status === 'urgent' ? status : null);
</script>

<Card title="The Wall" subtitle="Spent against limit, last {MONTHS} months" class="mt-4">
  {#snippet actions()}
    {#if error}
      <button type="button" class="btn-ghost text-xs" onclick={() => load(month)}>Retry</button>
    {/if}
  {/snippet}
  {#if error}
    <p class="text-sm text-rose-600 dark:text-rose-400">{error}</p>
  {:else if loading && !history}
    <p class="muted text-sm">Loading…</p>
  {:else if history && rows.length}
    <div class="-mx-5 overflow-x-auto px-5">
      <table class="w-full min-w-[40rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs">
            <th class="py-1 pr-3 font-medium">Category</th>
            {#each history.months as m (m)}
              <th class="px-1 py-1 text-center font-medium">{formatMonth(m)}</th>
            {/each}
          </tr>
        </thead>
        <tbody>
          {#each rows as row (row.category)}
            <tr>
              <td class="py-1 pr-3 font-medium whitespace-nowrap">{row.category}</td>
              {#each row.cells as cell (cell.month)}
                {@const flag = flagged(cell.status)}
                <td class="px-1 py-1">
                  <div class="rounded-lg px-2 py-1.5 text-center text-xs {wash[cell.status]}">
                    {#if cell.status === 'empty'}
                      <span class="muted">-</span>
                    {:else}
                      <p class="flex items-center justify-center gap-1 font-semibold">
                        {#if flag}<Icon name={STATUS[statusTone[flag]].icon} size={12} /><span class="sr-only"
                            >{statusLabel[flag]}:</span
                          >{/if}
                        {cell.percent === null ? 'No limit' : formatPercent(cell.percent, 0)}
                      </p>
                      <Money value={cell.spent} class="muted text-[11px]" />
                    {/if}
                  </div>
                </td>
              {/each}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted mt-3 text-xs">
      Green under 70% of the limit, amber with a warning sign from 70%, red with a stop sign from 90%, violet for
      spending without a limit.
    </p>
  {:else}
    <EmptyState title="No budget history" detail="Limits and spending of the last months show here." />
  {/if}
</Card>
