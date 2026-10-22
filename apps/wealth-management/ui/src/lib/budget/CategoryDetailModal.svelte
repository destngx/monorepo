<script lang="ts">
  import type { BudgetLine, SpendingWindows, Transaction } from '$lib/api/types';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Donut from '$lib/charts/Donut.svelte';
  import { DAY_TO_DAY, EXPENSE, INCOME, SPLIT_SECOND, SPLIT_SHARE } from '$lib/charts/palette';
  import Badge from '$lib/components/Badge.svelte';
  import { Modal, SkeletonChart, SkeletonTable } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatDate, formatMonth, formatPercent, todayISO } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { categoryTransactions, dailySplitSpend, detailEnd, rowSplitTag, splitSeries } from './detail';
  import type { CategoryHandler } from './status';

  // Category detail (WM-BUD-4).
  interface Props {
    line: BudgetLine | null;
    month: string;
    /** Null while the transactions are still loading. */
    transactions: Transaction[] | null;
    /** The last 30 days and 3 months (WM-BUD-12); null while loading or failed. */
    spending?: SpendingWindows | null;
    onclose: () => void;
    /** Starts moving limit away from this category. */
    onmove: CategoryHandler;
  }

  let { line, month, transactions, spending = null, onclose, onmove }: Props = $props();

  // Split tags (WM-CAT-3), e.g. eating out inside a food category: the month's share from the
  // report, the last 30 days and 3 months from the spending windows, and the daily chart stacked.
  const splitTags = $derived(line?.splits?.map((split) => split.tag) ?? []);
  const detailRows = $derived(line && transactions ? categoryTransactions(transactions, line.category, month) : []);
  const detailDays = $derived(
    line && transactions ? dailySplitSpend(transactions, line.category, detailEnd(month, todayISO()), splitTags) : [],
  );
  const PART_COLOR = { rest: DAY_TO_DAY, first: SPLIT_SHARE, second: SPLIT_SECOND } as const;
  const detailSeries = $derived(
    splitSeries(detailDays, splitTags).map(({ name, part, values }) => ({
      name,
      values,
      color: splitTags.length ? PART_COLOR[part] : EXPENSE,
    })),
  );
  const windowSplits = $derived.by(() => {
    const key = line?.category.trim().toLowerCase();
    const find = (index: number) =>
      spending?.windows[index]?.categories.find((category) => category.category.trim().toLowerCase() === key);
    return { recent: find(0), months: find(1), monthsDays: spending?.windows[1]?.days ?? 90 };
  });
  const splitOf = (category: { splits?: { tag: string; spent: number }[] } | undefined, tag: string) =>
    category?.splits?.find((split) => split.tag === tag)?.spent ?? 0;
</script>

<Modal open={line !== null} title={line?.category ?? ''} {onclose} wide>
  {#if line}
    <div class="grid grid-cols-1 items-center gap-6 sm:grid-cols-[auto_1fr]">
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
    {#if line.splits?.length}
      <h3 class="mt-6 mb-2 text-sm font-semibold">Split tags</h3>
      <div class="overflow-x-auto">
        <table class="w-full text-sm" data-testid="detail-splits">
          <thead class="muted text-left text-xs">
            <tr>
              <th class="py-1.5 font-medium">Tag</th>
              <th class="py-1.5 text-right font-medium">{formatMonth(month)}</th>
              <th class="py-1.5 text-right font-medium">Last 30 days</th>
              <th class="py-1.5 text-right font-medium">3 months, a month</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
            {#each line.splits as split (split.tag)}
              <tr>
                <td class="py-1.5 font-medium">#{split.tag}</td>
                <td class="tabular py-1.5 text-right"
                  ><Money value={split.spent} />
                  <span class="muted text-xs">({formatPercent(split.percentOfSpent, 0)})</span></td
                >
                <td class="tabular py-1.5 text-right">
                  {#if spending}<Money value={splitOf(windowSplits.recent, split.tag)} />{:else}<span class="muted"
                      >-</span
                    >{/if}
                </td>
                <td class="tabular py-1.5 text-right">
                  {#if spending}<Money
                      value={Math.round((splitOf(windowSplits.months, split.tag) / windowSplits.monthsDays) * 30.4)}
                    />{:else}<span class="muted">-</span>{/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    <h3 class="mt-6 mb-2 text-sm font-semibold">Daily spend, last 30 days</h3>
    {#if transactions === null}
      <SkeletonChart height={160} label="daily spend" />
    {:else}<BarChart
        labels={detailDays.map((d, i) => (i % 5 === 0 ? d.date.slice(8, 10) : ''))}
        series={detailSeries}
        stacked
        height={160}
        label="Daily spend"
      />{/if}
    <h3 class="mt-6 mb-2 text-sm font-semibold">Transactions in {formatMonth(month)}</h3>
    {#if transactions === null}
      <SkeletonTable rows={4} columns={2} label="transactions" />
    {:else if detailRows.length}
      <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5">
        {#each detailRows as tx (tx.id)}
          {@const tag = rowSplitTag(tx, splitTags)}
          <li class="flex justify-between gap-3 py-2">
            <span
              ><span class="muted tabular mr-2">{formatDate(tx.date)}</span>{tx.payee}
              {#if tag}<Badge tone="violet">#{tag}</Badge>{/if}</span
            >
            <Money value={(tx.deposit ?? 0) - (tx.payment ?? 0)} tone />
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No transactions this month.</p>
    {/if}
    {#if line.limit > 0 && line.categoryType === 'expense' && !line.lending}
      {@const from = line.category}
      <button type="button" class="btn-ghost mt-4" onclick={() => onmove(from)}>Move money from {from}</button>
    {/if}
  {/if}
</Modal>
