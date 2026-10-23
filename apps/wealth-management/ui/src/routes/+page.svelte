<script lang="ts">
  import { base } from '$app/paths';
  import BriefingCard from '$lib/ai/BriefingCard.svelte';
  import ChartInsight from '$lib/ai/ChartInsight.svelte';
  import { track, Button, Card, Deferred, Icon, SkeletonCard, SkeletonChart, SkeletonHeadline } from '@libs/ui-kit';
  import type { DashboardSummary } from '$lib/api/types';
  import SafeToSpendCard from '$lib/bills/SafeToSpendCard.svelte';
  import BarChart from '$lib/charts/BarChart.svelte';
  import { EXPENSE, INCOME } from '$lib/charts/palette';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import MonthSavingsCard from '$lib/home/MonthSavingsCard.svelte';
  import NetWorthCard from '$lib/home/NetWorthCard.svelte';
  import NeedsYouCard from '$lib/home/NeedsYouCard.svelte';
  import { formatDate, formatMonth } from '$lib/format';
  import type { LoadedRates } from '$lib/rates';

  // Home (ROADMAP R5, P2): what needs me first, then safe-to-spend, the net
  // worth trend and this month's savings, then a short briefing; the
  // income vs expense history is the detail at the bottom. Each part streams
  // in on its own, so the slowest call never holds the page.
  let { data } = $props();
  const summary = track(() => data.summary);
  const usd = track(() => data.usd);
  // The USD rate never fails the card: a failure reads as "USD value unavailable".
  const usdRates = $derived<LoadedRates | null>(
    usd.value ?? (usd.failure ? { rates: { VND: 1 }, quotes: {}, errors: [usd.failure.message] } : null),
  );

  type Range = 'ytd' | '6m' | '12m';
  let range = $state<Range>('6m');

  function historyOf(value: DashboardSummary) {
    const all = value.cashFlowHistory;
    if (range === '12m') return all;
    if (range === '6m') return all.slice(-6);
    const year = value.asOf.slice(0, 4);
    return all.filter((month) => month.month.startsWith(year));
  }
  const history = $derived(summary.value ? historyOf(summary.value) : []);
</script>

<PageHeader title="Dashboard" subtitle={summary.value ? `As of ${formatDate(summary.value.asOf)}` : undefined}>
  {#snippet actions()}
    <!-- WM-DASH-7: quick action; sync lives in the app header -->
    <a class="btn-ghost" href="{base}/transactions/capture"><Icon name="camera" size={16} /> Smart capture</a>
  {/snippet}
</PageHeader>

<!-- WM-HOME-1: the headline answer -->
<Deferred data={data.needsYou} label="what needs you">
  {#snippet children(inbox)}<NeedsYouCard {inbox} />{/snippet}
  {#snippet skeleton()}
    <SkeletonHeadline question="What needs me?" label="what needs you" />
  {/snippet}
  {#snippet error(failure, retry)}
    <PageHeadline
      question="What needs me?"
      tone="neutral"
      title="Could not check what needs you"
      id="needs-you-headline"
    >
      {#snippet actions()}<Button size="sm" icon="refresh" onclick={retry}>Retry</Button>{/snippet}
      <p class="muted">{failure.message}</p>
    </PageHeadline>
  {/snippet}
</Deferred>

<!-- WM-HOME-2: the evidence -->
<div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-3">
  <Deferred data={data.safeToSpend} label="safe to spend">
    {#snippet children(value)}<SafeToSpendCard summary={value} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Safe to spend" lines={3} />{/snippet}
  </Deferred>
  <Deferred data={data.summary} label="net worth">
    {#snippet children(value)}<NetWorthCard summary={value} usd={usdRates} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Net worth" lines={2} chart={64} />{/snippet}
  </Deferred>
  <Deferred data={data.summary} label="savings this month">
    {#snippet children(value)}<MonthSavingsCard summary={value} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Savings this month" lines={4} />{/snippet}
  </Deferred>
</div>

<!-- WM-DASH-6 -->
<div class="mt-4"><BriefingCard compact /></div>

<!-- WM-DASH-5: the detail -->
<Card title="Income vs expense" class="mt-4">
  {#snippet actions()}
    <a href="{base}/cash-flow" class="link text-xs">Yearly view</a>
    {#if summary.value}
      <ChartInsight
        chartType="bar"
        title="Income vs expense"
        timeframe={range}
        data={history.map((m) => ({ month: m.month, income: m.income, expense: m.expense }))}
      />
    {/if}
    <div class="flex rounded-xl bg-stone-900/5 p-0.5 text-xs dark:bg-white/5" role="group" aria-label="Range">
      {#each [['ytd', 'YTD'], ['6m', '6M'], ['12m', '12M']] as [value, text] (value)}
        <button
          type="button"
          aria-pressed={range === value}
          class="rounded-lg px-2.5 py-1 transition {range === value
            ? 'bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900'
            : 'muted hover:bg-white/60 hover:text-stone-800 dark:hover:bg-white/10 dark:hover:text-stone-100'}"
          onclick={() => (range = value as Range)}>{text}</button
        >
      {/each}
    </div>
  {/snippet}
  <Deferred data={data.summary} label="income and expense">
    {#snippet children()}
      <BarChart
        labels={history.map((m) => formatMonth(m.month, true))}
        series={[
          { name: 'Income', color: INCOME, values: history.map((m) => m.income) },
          { name: 'Expense', color: EXPENSE, values: history.map((m) => m.expense) },
        ]}
        label="Monthly income and expense"
      />
    {/snippet}
    {#snippet skeleton()}<SkeletonChart label="income and expense" />{/snippet}
  </Deferred>
</Card>
