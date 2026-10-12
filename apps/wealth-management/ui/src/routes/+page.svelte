<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import BriefingCard from '$lib/ai/BriefingCard.svelte';
  import ChartInsight from '$lib/ai/ChartInsight.svelte';
  import { accountIcon, creditRemaining, isCreditAccount } from '$lib/accounts';
  import BarChart from '$lib/charts/BarChart.svelte';
  import { EXPENSE, INCOME } from '$lib/charts/palette';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { budgetBarClass, trendDelta, vndToUsd } from '$lib/dashboard';
  import { formatDate, formatMonth, formatPercent, formatSignedPercent } from '$lib/format';
  import { sync } from '$lib/state/sync.svelte';

  let { data } = $props();
  const summary = $derived(data.summary);

  // WM-DASH-7: USD view and an explicit 30-day delta.
  const netWorthUsd = $derived(vndToUsd(summary.balanceSheet.netWorth, data.usd.rates.USD));
  const delta30 = $derived(trendDelta(summary.netWorthTrend));

  const quickActions = [
    { href: '/transactions?new=1', label: 'Add transaction', icon: 'plus' },
    { href: '/transactions/capture', label: 'Smart capture', icon: 'camera' },
    { href: '/budget', label: 'Budget', icon: 'budget' },
  ] as const;

  type Range = 'ytd' | '6m' | '12m';
  let range = $state<Range>('6m');

  const history = $derived.by(() => {
    const all = summary.cashFlowHistory;
    if (range === '12m') return all;
    if (range === '6m') return all.slice(-6);
    const year = summary.asOf.slice(0, 4);
    return all.filter((month) => month.month.startsWith(year));
  });

  const topBudget = $derived(
    data.budget.lines
      .filter((line) => line.categoryType === 'expense' && line.limit > 0)
      .sort((a, b) => b.percentUsed - a.percentUsed)
      .slice(0, 5),
  );
</script>

<PageHeader title="Dashboard" subtitle="As of {formatDate(summary.asOf)}" />

<div class="grid gap-4 lg:grid-cols-3">
  <!-- WM-DASH-1 -->
  <Card title="Net worth" class="lg:col-span-1">
    <div class="text-3xl font-semibold tracking-tight"><Money value={summary.balanceSheet.netWorth} /></div>
    <p class="muted mt-0.5 text-sm">
      {#if netWorthUsd !== null}
        ≈ <Money value={netWorthUsd} currency="USD" />
      {:else}
        <span title={data.usd.errors.join('; ')}>USD value unavailable</span>
      {/if}
    </p>
    <p class="mt-1 text-sm">
      {#if summary.netWorthChangePercent !== null}
        <span
          class={summary.netWorthChangePercent >= 0
            ? 'text-emerald-600 dark:text-emerald-400'
            : 'text-orange-600 dark:text-orange-400'}>{formatSignedPercent(summary.netWorthChangePercent)}</span
        >
        <span class="muted">vs last month</span>
      {/if}
    </p>
    <div class="mt-4">
      <Sparkline values={summary.netWorthTrend.map((p) => p.value)} width={320} height={64} label="30-day net worth" />
    </div>
    {#if delta30}
      <p class="mt-2 text-sm">
        <span class="muted">30-day change:</span>
        <Money value={delta30.delta} signed tone />
        {#if delta30.percent !== null}<span class="muted">({formatSignedPercent(delta30.percent)})</span>{/if}
      </p>
    {/if}
    <p class="muted mt-3 text-sm">
      Net growth this month: <Money value={summary.netGrowthThisMonth} signed tone />
    </p>
  </Card>

  <!-- WM-DASH-2 -->
  <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:col-span-2">
    <Stat label="Total assets" value={summary.balanceSheet.assets} />
    <Stat label="Total liabilities" value={summary.balanceSheet.liabilities} />
    <Stat label="Income" value={summary.thisMonth.income} change={summary.incomeChangePercent} hint=" vs last month" />
    <Stat
      label="Expense"
      value={summary.thisMonth.expense}
      change={summary.expenseChangePercent}
      invert
      hint=" vs last month"
    />
    <Stat label="Cash flow">
      <Money value={summary.thisMonth.cashFlow} signed tone />
    </Stat>
    <Stat label="Savings rate">
      <span class="tabular">{formatPercent(summary.thisMonth.savingsRate)}</span>
    </Stat>
  </div>
</div>

<!-- WM-DASH-7: Quick Action hub -->
<nav class="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4" aria-label="Quick actions">
  {#each quickActions as action (action.href)}
    <a href="{base}{action.href}" class="glass flex items-center gap-3 p-3 text-sm font-medium hover:bg-indigo-500/5">
      <span class="grid size-8 place-items-center rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400"
        ><Icon name={action.icon} size={16} /></span
      >
      {action.label}
    </a>
  {/each}
  <button
    type="button"
    class="glass flex items-center gap-3 p-3 text-left text-sm font-medium hover:bg-indigo-500/5 disabled:opacity-60"
    disabled={sync.syncing}
    onclick={() => sync.run(invalidateAll)}
  >
    <span class="grid size-8 place-items-center rounded-lg bg-indigo-500/10 text-indigo-600 dark:text-indigo-400"
      ><Icon name="refresh" size={16} class={sync.syncing ? 'animate-spin' : ''} /></span
    >
    {sync.syncing ? 'Syncing…' : 'Sync now'}
  </button>
</nav>

<div class="mt-4 grid gap-4 lg:grid-cols-5">
  <!-- WM-DASH-5 -->
  <Card title="Income vs expense" class="lg:col-span-3">
    {#snippet actions()}
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

  <!-- WM-DASH-3 -->
  <Card title="Accounts" class="lg:col-span-2">
    {#snippet actions()}
      <a href="{base}/accounts" class="text-xs text-indigo-600 hover:underline dark:text-indigo-400">View all</a>
    {/snippet}
    <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
      {#each summary.topAccounts as account (account.name)}
        <li class="flex items-center gap-3 py-2.5">
          <span class="grid size-8 place-items-center rounded-lg bg-slate-900/5 dark:bg-white/5"
            ><Icon name={accountIcon(account.type)} size={16} /></span
          >
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium">{account.name}</p>
            <p class="muted text-xs">{account.type}</p>
          </div>
          <div class="text-right text-sm">
            <Money value={account.balance} currency={account.currency} />
            {#if isCreditAccount(account)}
              <p class="muted text-xs">Available <Money value={creditRemaining(account)} /></p>
            {/if}
          </div>
        </li>
      {/each}
    </ul>
  </Card>
</div>

<!-- WM-DASH-4 -->
<Card title="Budget this month" subtitle="Top expense categories by % used" class="mt-4">
  {#snippet actions()}
    <a href="{base}/budget" class="text-xs text-indigo-600 hover:underline dark:text-indigo-400">Open budget</a>
  {/snippet}
  {#if topBudget.length}
    <ul class="space-y-3">
      {#each topBudget as line (line.category)}
        <li>
          <div class="mb-1 flex justify-between gap-3 text-sm">
            <span class="truncate">{line.category}</span>
            <span class="muted tabular whitespace-nowrap"
              ><Money value={line.spent} /> / <Money value={line.limit} /> · {formatPercent(line.percentUsed, 0)}</span
            >
          </div>
          <!-- WM-RULE-6: engine status bands (D4 70/90) -->
          <ProgressBar percent={line.percentUsed} color={budgetBarClass(line)} label={line.category} />
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted text-sm">No expense limits set for this month.</p>
  {/if}
</Card>

<!-- WM-DASH-6 -->
<div class="mt-4"><BriefingCard /></div>
