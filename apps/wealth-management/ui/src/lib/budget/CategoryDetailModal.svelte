<script lang="ts">
  import type { BudgetLine, Transaction } from '$lib/api/types';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Donut from '$lib/charts/Donut.svelte';
  import { EXPENSE, INCOME } from '$lib/charts/palette';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate, formatMonth, formatPercent, todayISO } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { categoryTransactions, dailySpend, detailEnd } from './detail';
  import type { CategoryHandler } from './status';

  // Category detail (WM-BUD-4).
  interface Props {
    line: BudgetLine | null;
    month: string;
    transactions: Transaction[];
    onclose: () => void;
    /** Starts moving limit away from this category. */
    onmove: CategoryHandler;
  }

  let { line, month, transactions, onclose, onmove }: Props = $props();

  const detailRows = $derived(line ? categoryTransactions(transactions, line.category, month) : []);
  const detailDays = $derived(line ? dailySpend(transactions, line.category, detailEnd(month, todayISO())) : []);
</script>

<Modal open={line !== null} title={line?.category ?? ''} {onclose} wide>
  {#if line}
    <div class="grid items-center gap-6 sm:grid-cols-[auto_1fr]">
      <Donut
        items={[
          { label: 'Spent', value: line.spent, color: EXPENSE },
          { label: 'Remaining', value: line.remaining, color: INCOME },
        ]}
        size={160}
        label="Spent vs remaining"
      >
        {#snippet center()}
          <span class="text-lg font-semibold">{formatPercent(line?.percentUsed, 0)}</span>
          <span class="muted text-xs">used</span>
        {/snippet}
      </Donut>
      <dl class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <dt class="muted text-xs">Spent</dt>
          <dd><Money value={line.spent} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Limit</dt>
          <dd><Money value={line.limit} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Year spent</dt>
          <dd><Money value={line.yearlySpent} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Year remaining</dt>
          <dd><Money value={line.yearlyRemaining} /></dd>
        </div>
        {#if line.burnOutDate}<div class="col-span-2">
            <dt class="muted text-xs">Runs out on</dt>
            <dd class={STATUS.serious.ink}>{formatDate(line.burnOutDate)} at the current pace</dd>
          </div>{/if}
        {#if line.note}<div class="col-span-2">
            <dt class="muted text-xs">Note</dt>
            <dd>{line.note}</dd>
          </div>{/if}
      </dl>
    </div>
    <h3 class="mt-6 mb-2 text-sm font-semibold">Daily spend, last 30 days</h3>
    <BarChart
      labels={detailDays.map((d, i) => (i % 5 === 0 ? d.date.slice(8, 10) : ''))}
      series={[{ name: 'Spend', color: EXPENSE, values: detailDays.map((d) => d.value) }]}
      height={160}
      label="Daily spend"
    />
    <h3 class="mt-6 mb-2 text-sm font-semibold">Transactions in {formatMonth(month)}</h3>
    {#if detailRows.length}
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each detailRows as tx (tx.id)}
          <li class="flex justify-between gap-3 py-2">
            <span><span class="muted tabular mr-2">{formatDate(tx.date)}</span>{tx.payee}</span>
            <Money value={(tx.deposit ?? 0) - (tx.payment ?? 0)} tone />
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No transactions this month.</p>
    {/if}
    {#if line.limit > 0 && line.categoryType === 'expense'}
      {@const from = line.category}
      <button type="button" class="btn-ghost mt-4" onclick={() => onmove(from)}>Move money from {from}</button>
    {/if}
  {/if}
</Modal>
