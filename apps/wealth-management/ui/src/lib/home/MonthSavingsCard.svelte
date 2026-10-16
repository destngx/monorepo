<script lang="ts">
  import type { DashboardSummary } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { expensePaceView, sameDaysHint, savingsRateTile } from '$lib/dashboard-month';
  import { formatDate, formatMonth, formatPercent, formatSignedPercent } from '$lib/format';

  // WM-DASH-2/9: this month to date against the same days of last month,
  // and where expense is heading by the month end.
  let { summary }: { summary: DashboardSummary } = $props();

  const compareHint = $derived(sameDaysHint(summary));
  const savings = $derived(savingsRateTile(summary.thisMonth, summary.lastMonthToDate));
  const pace = $derived(expensePaceView(summary.expensePace));
</script>

{#snippet change(percent: number | null, invert: boolean)}
  {#if percent !== null}
    <span
      class="text-xs {(invert ? percent < 0 : percent > 0)
        ? 'text-emerald-600 dark:text-emerald-400'
        : 'text-orange-600 dark:text-orange-400'}">{formatSignedPercent(percent)}</span
    >
  {/if}
{/snippet}

<Card title="Savings this month" subtitle="Month to date,{compareHint}">
  <p class="text-3xl font-semibold tracking-tight tabular" data-testid="savings-rate">{formatPercent(savings.rate)}</p>
  <p class="muted mt-0.5 text-sm">Savings rate, {savings.hint}</p>
  <dl class="mt-3 grid grid-cols-[1fr_auto] items-baseline gap-x-4 gap-y-0.5 text-sm">
    <dt class="muted">Income</dt>
    <dd class="text-right">
      <Money value={summary.thisMonth.income} />
      {@render change(summary.incomeChangePercent, false)}
    </dd>
    <dt class="muted">Expense</dt>
    <dd class="text-right">
      <Money value={summary.thisMonth.expense} />
      {@render change(summary.expenseChangePercent, true)}
    </dd>
    <dt class="muted">Cash flow</dt>
    <dd class="text-right"><Money value={summary.thisMonth.cashFlow} signed tone /></dd>
  </dl>
  {#if pace}
    <p class="mt-3 text-sm" data-testid="expense-pace">
      {#if pace.kind === 'pace'}
        <span class="muted">Expense on pace for</span>
        <span class="font-semibold"><Money value={pace.projected} /></span>
        <span class="muted">by {formatDate(pace.monthEnd)}, vs</span>
        <Money value={pace.lastMonthExpense} />
        <span class="muted">in {formatMonth(pace.lastMonth)}</span>
        {@render change(pace.changePercent, true)}
        {#if pace.fixed > 0}
          <span class="muted block text-xs">Fixed bills are counted once; the rest is extrapolated by day.</span>
        {/if}
      {:else}
        <span class="muted">Expense pace shows from day {pace.fromDay};</span>
        <Money value={pace.spent} />
        <span class="muted">spent so far.</span>
      {/if}
    </p>
  {/if}
</Card>
