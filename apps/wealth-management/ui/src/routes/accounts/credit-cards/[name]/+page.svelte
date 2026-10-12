<script lang="ts">
  import { base } from '$app/paths';
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { api } from '$lib/api/endpoints';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Ring from '$lib/charts/Ring.svelte';
  import { UTILIZATION_COLOR } from '$lib/cards/cards';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { currentMonth, formatDate, formatMonth, formatPercent } from '$lib/format';
  import CategoryBadge from '$lib/transactions/CategoryBadge.svelte';
  import TagChip from '$lib/transactions/TagChip.svelte';

  let { data } = $props();
  const card = $derived(data.card);
  const cycle = $derived(card.cycle);

  // AI summary for the current or a past month (WM-CARD-5).
  const summaryMonths = $derived(
    [...new Set([currentMonth(), ...card.history.map((m) => m.month)])].sort().reverse().slice(0, 12),
  );
  let summaryMonth = $state(currentMonth());

  type Tab = 'cycle' | 'history' | 'rules';
  let tab = $state<Tab>('cycle');

  // History filters (WM-CARD-2): one year at a time, oldest first for the chart.
  const years = $derived([...new Set(card.history.map((m) => m.month.slice(0, 4)))].sort().reverse());
  let year = $state<string | null>(null);
  const activeYear = $derived(year ?? years[0] ?? '');
  const yearHistory = $derived(
    card.history.filter((m) => m.month.startsWith(activeYear)).sort((a, b) => a.month.localeCompare(b.month)),
  );
  const ytd = $derived(
    yearHistory.reduce(
      (total, m) => ({
        spent: total.spent + m.spent,
        potential: total.potential + m.potentialCashback,
        refund: total.refund + m.actualRefund,
        missed: total.missed + m.missed,
      }),
      { spent: 0, potential: 0, refund: 0, missed: 0 },
    ),
  );
</script>

<a href="{base}/accounts/credit-cards" class="muted mb-3 inline-flex items-center gap-1 text-sm hover:underline"
  ><Icon name="chevronLeft" size={14} /> Credit cards</a
>
<PageHeader
  title={card.name}
  subtitle="{card.bank} · {card.accountName}{card.expiry ? ` · expires ${card.expiry}` : ''}"
/>

<div class="grid gap-4 lg:grid-cols-3">
  <!-- WM-CARD-4 -->
  <Card title="Utilisation">
    <div class="flex items-center gap-5">
      <Ring percent={card.utilization} color={UTILIZATION_COLOR[card.utilizationBand]} label="Utilisation">
        <span class="text-xl font-semibold">{formatPercent(card.utilization, 0)}</span>
        <span class="muted text-xs">used</span>
      </Ring>
      <dl class="space-y-2 text-sm">
        <div>
          <dt class="muted text-xs">Used</dt>
          <dd><Money value={card.usage} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Remaining</dt>
          <dd><Money value={card.remainingLimit} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Limit</dt>
          <dd><Money value={card.limit} /></dd>
        </div>
      </dl>
    </div>
  </Card>

  <Card title="Statement cycle">
    <p class="text-sm">{formatDate(cycle.cycleStart)} - {formatDate(cycle.cycleEnd)}</p>
    <div class="mt-3"><ProgressBar percent={cycle.cycleProgress} color="bg-indigo-500" label="Cycle progress" /></div>
    <p class="muted mt-1 text-xs">
      Day {cycle.cycleElapsedDays} of {cycle.cycleDays} · statement day {cycle.statementDay}
    </p>
    <p class="mt-4 text-sm">Spent this cycle: <Money value={card.spentThisCycle} /></p>
    <p class="text-sm">Estimated cashback: <Money value={card.estimatedCashback} /></p>
  </Card>

  <Card title="Payment due">
    <div class="flex items-baseline gap-2">
      <span class="text-3xl font-semibold {cycle.dueUrgent ? 'text-rose-600 dark:text-rose-400' : ''}"
        >{cycle.daysUntilDue}</span
      >
      <span class="muted text-sm">days until {formatDate(cycle.nextDueDate)}</span>
    </div>
    {#if cycle.dueUrgent}<div class="mt-2"><Badge tone="red">Pay soon</Badge></div>{/if}
    <dl class="mt-4 grid grid-cols-2 gap-3 text-sm">
      <div>
        <dt class="muted text-xs">Outstanding</dt>
        <dd><Money value={cycle.outstandingAmount} /></dd>
      </div>
      <div>
        <dt class="muted text-xs">Minimum (5%)</dt>
        <dd><Money value={cycle.minimumPayment} /></dd>
      </div>
    </dl>
  </Card>
</div>

<div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Stat label="Refunds YTD" value={card.refundYtd} />
  <Stat label="Annual fee" value={card.annualFee} />
  <Stat label="Net earn YTD"><Money value={card.netEarnYtd} tone signed /></Stat>
  <Stat label="Lifetime fees" value={card.lifetimeFees} />
</div>

<div class="mt-6 mb-4 flex gap-1 rounded-xl bg-slate-900/5 p-1 text-sm dark:bg-white/5" role="tablist">
  {#each [['cycle', 'This cycle'], ['history', 'History'], ['rules', 'Rules']] as [value, label] (value)}
    <button
      type="button"
      role="tab"
      aria-selected={tab === value}
      class="flex-1 rounded-lg px-3 py-1.5 {tab === value
        ? 'bg-white font-medium shadow-sm dark:bg-slate-700'
        : 'muted'}"
      onclick={() => (tab = value as Tab)}>{label}</button
    >
  {/each}
</div>

{#if tab === 'cycle'}
  <Card title="Transactions this cycle" subtitle="{card.cycleTransactions.length} transactions">
    {#if card.cycleTransactions.length}
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each card.cycleTransactions as tx (tx.id)}
          <li class="flex flex-wrap items-center justify-between gap-3 py-2.5">
            <div>
              <p>
                <span class="muted tabular mr-2">{formatDate(tx.date)}</span><span class="font-medium">{tx.payee}</span>
              </p>
              <div class="mt-1 flex flex-wrap gap-1">
                <CategoryBadge name={tx.category} type={tx.categoryType} />
                {#each tx.tags as tag (tag)}<TagChip {tag} />{/each}
              </div>
            </div>
            <Money value={(tx.deposit ?? 0) - (tx.payment ?? 0)} tone />
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No transactions in the current cycle.</p>
    {/if}
  </Card>
{:else if tab === 'history'}
  <Card title="Cashback efficiency">
    {#snippet actions()}
      <select
        class="field w-auto py-1"
        aria-label="Year"
        value={activeYear}
        onchange={(e) => (year = e.currentTarget.value)}
      >
        {#each years as y (y)}<option value={y}>{y}</option>{/each}
      </select>
    {/snippet}
    <div class="mb-4 grid grid-cols-2 gap-3 text-sm sm:grid-cols-4">
      <div>
        <p class="muted text-xs">Spent</p>
        <Money value={ytd.spent} />
      </div>
      <div>
        <p class="muted text-xs">Potential cashback</p>
        <Money value={ytd.potential} />
      </div>
      <div>
        <p class="muted text-xs">Actual refund</p>
        <Money value={ytd.refund} />
      </div>
      <div>
        <p class="muted text-xs">Missed</p>
        <Money value={ytd.missed} />
      </div>
    </div>
    <BarChart
      labels={yearHistory.map((m) => formatMonth(m.month, true))}
      series={[
        { name: 'Earned', color: '#10b981', values: yearHistory.map((m) => m.actualRefund) },
        { name: 'Missed', color: '#f97316', values: yearHistory.map((m) => m.missed) },
        { name: 'Potential', color: '#6366f1', values: yearHistory.map((m) => m.potentialCashback) },
      ]}
      label="Cashback earned vs missed"
    />
    <div class="-mx-5 mt-4 overflow-x-auto">
      <table class="w-full text-sm">
        <thead
          ><tr class="muted text-left text-xs uppercase"
            ><th class="px-5 py-2">Month</th><th class="px-5 py-2 text-right">Spent</th><th class="px-5 py-2 text-right"
              >Potential</th
            ><th class="px-5 py-2 text-right">Refund</th><th class="px-5 py-2 text-right">Efficiency</th></tr
          ></thead
        >
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each [...yearHistory].reverse() as m (m.month)}
            <tr>
              <td class="px-5 py-2">{formatMonth(m.month)}</td>
              <td class="px-5 py-2 text-right"><Money value={m.spent} /></td>
              <td class="px-5 py-2 text-right"><Money value={m.potentialCashback} /></td>
              <td class="px-5 py-2 text-right"><Money value={m.actualRefund} /></td>
              <td class="tabular px-5 py-2 text-right">{formatPercent(m.efficiency, 0)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </Card>
{:else}
  <Card title="Cashback rules" subtitle="Tag transactions with the rule tag to earn its rate">
    <table class="w-full text-sm">
      <thead
        ><tr class="muted text-left text-xs uppercase"
          ><th class="py-2">Rule</th><th class="py-2">Tag</th><th class="py-2 text-right">Rate</th><th
            class="py-2 text-right">Monthly cap</th
          ></tr
        ></thead
      >
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each card.rules as rule (rule.name)}
          <tr>
            <td class="py-2"
              >{rule.name}
              {#if rule.fallback}<Badge>fallback</Badge>{/if}</td
            >
            <td class="py-2"
              >{#if rule.tag}<TagChip tag={rule.tag} />{:else}<span class="muted">-</span>{/if}</td
            >
            <td class="tabular py-2 text-right">{formatPercent(rule.rate, 1)}</td>
            <td class="py-2 text-right"
              >{#if rule.capMonthly > 0}<Money value={rule.capMonthly} />{:else}<span class="muted">No cap</span
                >{/if}</td
            >
          </tr>
        {/each}
      </tbody>
    </table>
  </Card>
{/if}

<!-- WM-CARD-5 -->
<div class="mt-4">
  <label class="label" for="summary-month">AI summary month</label>
  <select id="summary-month" class="field mb-3 w-auto" bind:value={summaryMonth}>
    {#each summaryMonths as month (month)}
      <option value={month}>{formatMonth(month)}</option>
    {/each}
  </select>
  <InsightCard
    title="AI monthly summary"
    subtitle="{card.name} · {formatMonth(summaryMonth)}"
    load={(force, signal) => api.review('credit-summary', { card: card.name, month: summaryMonth }, { force, signal })}
  />
</div>
