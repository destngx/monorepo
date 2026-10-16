<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import BriefingCard from '$lib/ai/BriefingCard.svelte';
  import ChartInsight from '$lib/ai/ChartInsight.svelte';
  import SafeToSpendCard from '$lib/bills/SafeToSpendCard.svelte';
  import BarChart from '$lib/charts/BarChart.svelte';
  import { EXPENSE, INCOME } from '$lib/charts/palette';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import MonthSavingsCard from '$lib/home/MonthSavingsCard.svelte';
  import NetWorthCard from '$lib/home/NetWorthCard.svelte';
  import NeedsYouCard from '$lib/home/NeedsYouCard.svelte';
  import { formatDate, formatMonth } from '$lib/format';
  import { sync } from '$lib/state/sync.svelte';

  // Home (ROADMAP R5, P2): what needs me first, then safe-to-spend, the net
  // worth trend and this month's savings, then a short briefing; the
  // income vs expense history is the detail at the bottom.
  let { data } = $props();
  const summary = $derived(data.summary);

  type Range = 'ytd' | '6m' | '12m';
  let range = $state<Range>('6m');

  const history = $derived.by(() => {
    const all = summary.cashFlowHistory;
    if (range === '12m') return all;
    if (range === '6m') return all.slice(-6);
    const year = summary.asOf.slice(0, 4);
    return all.filter((month) => month.month.startsWith(year));
  });
</script>

<PageHeader title="Dashboard" subtitle="As of {formatDate(summary.asOf)}">
  {#snippet actions()}
    <!-- WM-DASH-7: quick actions -->
    <a class="btn-ghost" href="{base}/transactions?new=1"><Icon name="plus" size={16} /> Add transaction</a>
    <a class="btn-ghost" href="{base}/transactions/capture"><Icon name="camera" size={16} /> Smart capture</a>
    <button type="button" class="btn-ghost" disabled={sync.syncing} onclick={() => sync.run(invalidateAll)}>
      <Icon name="refresh" size={16} class={sync.syncing ? 'animate-spin' : ''} />
      {sync.syncing ? 'Syncing…' : 'Sync now'}
    </button>
  {/snippet}
</PageHeader>

<!-- WM-HOME-1: the headline answer -->
{#if data.needsYou.ok}
  <NeedsYouCard inbox={data.needsYou.value} />
{:else}
  <PageHeadline question="What needs me?" tone="neutral" title="Could not check what needs you" id="needs-you-headline">
    <p class="muted">{data.needsYou.error}</p>
  </PageHeadline>
{/if}

<!-- WM-HOME-2: the evidence -->
<div class="mt-4 grid gap-4 lg:grid-cols-3">
  {#if data.safeToSpend.ok}
    <SafeToSpendCard summary={data.safeToSpend.value} />
  {:else}
    <Card title="Safe to spend">
      <p class="muted text-sm">Could not load safe-to-spend: {data.safeToSpend.error}</p>
    </Card>
  {/if}
  <NetWorthCard {summary} usd={data.usd} />
  <MonthSavingsCard {summary} />
</div>

<!-- WM-DASH-6 -->
<div class="mt-4"><BriefingCard compact /></div>

<!-- WM-DASH-5: the detail -->
<Card title="Income vs expense" class="mt-4">
  {#snippet actions()}
    <a href="{base}/cash-flow" class="text-xs text-indigo-600 hover:underline dark:text-indigo-400">Yearly view</a>
    <ChartInsight
      chartType="bar"
      title="Income vs expense"
      timeframe={range}
      data={history.map((m) => ({ month: m.month, income: m.income, expense: m.expense }))}
    />
    <div class="flex rounded-xl bg-slate-900/5 p-0.5 text-xs dark:bg-white/5" role="group" aria-label="Range">
      {#each [['ytd', 'YTD'], ['6m', '6M'], ['12m', '12M']] as [value, text] (value)}
        <button
          type="button"
          aria-pressed={range === value}
          class="rounded-lg px-2.5 py-1 {range === value ? 'bg-white shadow-sm dark:bg-slate-700' : 'muted'}"
          onclick={() => (range = value as Range)}>{text}</button
        >
      {/each}
    </div>
  {/snippet}
  <BarChart
    labels={history.map((m) => formatMonth(m.month, true))}
    series={[
      { name: 'Income', color: INCOME, values: history.map((m) => m.income) },
      { name: 'Expense', color: EXPENSE, values: history.map((m) => m.expense) },
    ]}
    label="Monthly income and expense"
  />
</Card>
