<script lang="ts">
  import { base } from '$app/paths';
  import AIExplain from '$lib/ai/AIExplain.svelte';
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { api } from '$lib/api/endpoints';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Ring from '$lib/charts/Ring.svelte';
  import {
    capsLeft,
    cardPeriodCap,
    cumulative,
    formatMccs,
    MCC_STATUS,
    RECONCILIATION,
    UTILIZATION,
  } from '$lib/cards/cards';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import MultiLineChart from '$lib/investments/fmarket/MultiLineChart.svelte';
  import { currentMonth, formatCompact, formatDate, formatMonth, formatPercent, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { STATUS } from '$lib/status';
  import CategoryBadge from '$lib/transactions/CategoryBadge.svelte';
  import TagChip from '$lib/transactions/TagChip.svelte';

  let { data } = $props();
  const bank = $derived(data.bank);
  const use = $derived(UTILIZATION[bank.utilizationBand]);
  const card = $derived(data.card);
  const cycle = $derived(card.cycle);
  const standing = $derived(card.standing);
  const statement = $derived(card.statement);
  // A programme without a card-level cap is still bounded by its rule and cap-group caps.
  const periodCaps = $derived(capsLeft(standing));

  // AI summary for the current or a past period (WM-CARD-5).
  const summaryMonths = $derived(
    [...new Set([currentMonth(), ...card.history.map((m) => m.month)])].sort().reverse().slice(0, 12),
  );
  let summaryMonth = $state(currentMonth());

  type Tab = 'period' | 'history' | 'rules';
  let tab = $state<Tab>('period');
  // A closed card has no current period: its history comes first.
  const tabs = $derived(
    (
      [
        ['period', 'This period'],
        ['history', 'History'],
        ['rules', 'Rules'],
      ] as const
    ).filter(([value]) => !card.closedFrom || value !== 'period'),
  );
  const activeTab = $derived<Tab>(card.closedFrom && tab === 'period' ? 'history' : tab);

  // History filters (WM-CARD-2): one year at a time, oldest first for the chart.
  const years = $derived([...new Set(card.history.map((m) => m.month.slice(0, 4)))].sort().reverse());
  let year = $state<string | null>(null);
  const activeYear = $derived(year ?? years[0] ?? '');
  const yearHistory = $derived(
    card.history.filter((m) => m.month.startsWith(activeYear)).sort((a, b) => a.month.localeCompare(b.month)),
  );
  const cap = $derived(cardPeriodCap(card));
  const barSeries = $derived([
    { name: 'Refund', color: '#10b981', values: yearHistory.map((m) => m.refund) },
    { name: 'Missed', color: '#f97316', values: yearHistory.map((m) => m.missed) },
    { name: 'Estimated', color: '#6366f1', values: yearHistory.map((m) => m.cashback.total) },
    ...(cap === null ? [] : [{ name: 'Period cap', color: '#94a3b8', values: yearHistory.map(() => cap) }]),
  ]);
  const cumulativeSeries = $derived([
    { name: 'Refund YTD', color: '#10b981', values: cumulative(yearHistory.map((m) => m.refund)) },
    { name: 'Estimated YTD', color: '#6366f1', values: cumulative(yearHistory.map((m) => m.cashback.total)) },
    ...(cap === null ? [] : [{ name: 'Cap YTD', color: '#94a3b8', values: cumulative(yearHistory.map(() => cap)) }]),
  ]);
  const formatMasked = (value: number) => (mask.revealed ? formatCompact(value) : MASK);

  const ytd = $derived(
    yearHistory.reduce(
      (total, m) => ({
        spent: total.spent + m.spent,
        estimated: total.estimated + m.cashback.total,
        refund: total.refund + m.refund,
        missed: total.missed + m.missed,
      }),
      { spent: 0, estimated: 0, refund: 0, missed: 0 },
    ),
  );
  const periodLabel = $derived(card.capPeriod === 'month' ? 'calendar month' : 'statement cycle');
</script>

<a href="{base}/accounts/credit-cards" class="muted mb-3 inline-flex items-center gap-1 text-sm hover:underline"
  ><Icon name="chevronLeft" size={14} /> Credit cards</a
>
<PageHeader
  title={card.name}
  subtitle="{card.bank} · {card.program} · {card.accountName}{card.last4 ? ` · *${card.last4}` : ''}{card.closedFrom
    ? ` · closed ${formatDate(card.closedFrom)}`
    : card.expiry
      ? ` · expires ${card.expiry}`
      : ''}"
/>

<div class="grid gap-4 lg:grid-cols-3">
  <!-- WM-CARD-16: the limit is the bank's, shared by its cards -->
  <Card title="{bank.name} limit" subtitle="Shared by the bank's cards">
    <div class="flex items-center gap-5">
      <div class="flex flex-col items-center gap-2">
        <Ring percent={bank.utilization} color={STATUS[use.tone].chart} label="Bank utilisation">
          <span class="text-xl font-semibold">{formatPercent(bank.utilization, 0)}</span>
          <span class="muted text-xs">used</span>
        </Ring>
        <Badge status={use.tone}>{use.label}</Badge>
      </div>
      <dl class="space-y-2 text-sm">
        <div>
          <dt class="muted text-xs">Used</dt>
          <dd><Money value={bank.usage} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Room</dt>
          <dd><Money value={bank.remainingLimit} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Limit</dt>
          <dd><Money value={bank.limit} /></dd>
        </div>
      </dl>
    </div>
  </Card>

  {#if card.closedFrom}
    <div class="lg:col-span-2">
      <Card title="Closed" subtitle="From {formatDate(card.closedFrom)}">
        <p class="text-sm">
          The card takes no payments from this date. Its earlier periods and the refunds paid for them stay in History;
          the Rules tab shows its last terms.
        </p>
      </Card>
    </div>
  {:else}
    <!-- WM-CARD-12 -->
    <Card title="This period" subtitle="Caps and tiers count per {periodLabel}">
      <p class="text-sm">{formatDate(standing.start)} - {formatDate(standing.end)}</p>
      <dl class="mt-3 grid grid-cols-2 gap-3 text-sm">
        <div>
          <dt class="muted text-xs">Spent</dt>
          <dd><Money value={standing.spent} /></dd>
        </div>
        <div>
          <dt class="muted flex items-center gap-1 text-xs">
            Cashback (est.)
            <!-- WM-CARD-17; keyed so another card starts a fresh explanation. -->
            {#key card.name}
              <AIExplain
                title="Cashback estimate"
                label="Explain the cashback estimate with AI"
                icon="info"
                load={(force, signal) => api.explainCashback({ card: card.name }, { force, signal })}
              />
            {/key}
          </dt>
          <dd><Money value={standing.cashback.total} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Confirmed</dt>
          <dd><Money value={standing.cashback.confirmed} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Cap left</dt>
          <dd>
            {#each periodCaps as capped (capped.name)}
              <p>
                <Money value={capped.left} />{#if standing.cap === 0}<span class="muted text-xs">
                    {capped.name}</span
                  >{/if}
              </p>
            {:else}<span class="muted">No cap</span>{/each}
          </dd>
        </div>
      </dl>
      {#each periodCaps as capped (capped.name)}
        <div class="mt-3">
          <ProgressBar
            percent={Math.min(100, ((capped.cap - capped.left) / capped.cap) * 100)}
            color="bg-emerald-500"
            label="{capped.name} cap used"
          />
        </div>
      {/each}
      {#if standing.nextTier}
        <p class="mt-3 text-xs">
          <Money value={standing.nextTier.spendNeeded} /> more qualifying spend reaches {standing.nextTier.rules.join(
            ', ',
          )}.
        </p>
      {/if}
      {#if standing.unconfirmedPayments > 0}
        <p class="muted mt-1 text-xs">
          {standing.unconfirmedPayments}
          {standing.unconfirmedPayments === 1 ? 'payment has' : 'payments have'} an assumed or unknown MCC, so part of the
          cashback is estimated. Confirm MCCs in the Merchants tab or tag payments mcc:NNNN.
        </p>
      {/if}
    </Card>

    <!-- WM-CARD-15 -->
    <Card title="Pay by">
      <div class="flex items-baseline gap-2">
        <span class="text-3xl font-semibold {cycle.payByUrgent ? STATUS.critical.ink : ''}">{cycle.daysUntilPayBy}</span
        >
        <span class="muted text-sm">days until {formatDate(cycle.payBy)}</span>
      </div>
      {#if cycle.payByUrgent}<div class="mt-2"><Badge status="critical">Pay soon</Badge></div>{/if}
      {#if statement}
        <div class="mt-4" data-testid="statement-estimate">
          <p class="muted text-xs">
            Estimated statement · {formatDate(statement.start)} - {formatDate(statement.end)}{statement.open
              ? ' · still open'
              : ''}
          </p>
          <p class="text-xl font-semibold"><Money value={statement.amount} /></p>
          <p class="muted mt-1 text-xs">
            Spent <Money value={statement.spent} />{#if statement.fees}
              + fees <Money value={statement.fees} />{/if}
          </p>
          <!-- A refund lowers the statement it is credited in; its memo month is the period it was earned in. -->
          {#each statement.refunds ?? [] as refund, i (i)}
            <p class="muted text-xs">
              - cashback <Money value={refund.amount} /> for {formatMonth(refund.for)}, credited {formatDate(
                refund.date,
              )}
            </p>
          {/each}
        </div>
      {/if}
      <p class="muted mt-3 text-xs">
        This card's share of the statement, from the ledger; payments already made are not deducted. Statement on day
        {cycle.statementDay}; the bank's due day is {cycle.dueDay}, paid by the 29th (28th in February) to stay clear of
        it.
      </p>
    </Card>
  {/if}
</div>

<div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Stat label="Refunds YTD" value={card.refundYtd} />
  <Stat label="Fees paid YTD" value={card.feesPaidYtd} hint="annual fee {formatMasked(card.annualFee)}" />
  <Stat label="Net earn YTD"><Money value={card.netEarnYtd} tone signed /></Stat>
  <Stat label="Lifetime fees" value={card.lifetimeFees} />
</div>

<div class="mt-6 mb-4 flex gap-1 rounded-xl bg-slate-900/5 p-1 text-sm dark:bg-white/5" role="tablist">
  {#each tabs as [value, label] (value)}
    <button
      type="button"
      role="tab"
      aria-selected={activeTab === value}
      class="flex-1 rounded-lg px-3 py-1.5 {activeTab === value
        ? 'bg-white font-medium shadow-sm dark:bg-slate-700'
        : 'muted'}"
      onclick={() => (tab = value as Tab)}>{label}</button
    >
  {/each}
</div>

{#if activeTab === 'period'}
  {#if standing.rules.length}
    <Card title="Cashback by rule" class="mb-4">
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each standing.rules as rule (rule.name)}
          <li class="flex flex-wrap items-center justify-between gap-3 py-2">
            <span
              >{rule.name}{#if rule.rules}<span class="muted text-xs">
                  · shared cap: {rule.rules.join(', ')}</span
                >{/if}</span
            >
            <span class="text-right">
              <Money value={rule.earned.total} />
              {#if rule.cap > 0}<span class="muted text-xs">
                  · <Money value={rule.capLeft} /> left of <Money value={rule.cap} /></span
                >{/if}
            </span>
          </li>
        {/each}
      </ul>
    </Card>
  {/if}
  <Card title="Transactions this period" subtitle="{card.transactions.length} transactions">
    {#if card.transactions.length}
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each card.transactions as { transaction: tx, mcc, rule, cashback } (tx.id)}
          <li class="flex flex-wrap items-center justify-between gap-3 py-2.5">
            <div>
              <p>
                <span class="muted tabular mr-2">{formatDate(tx.date)}</span><span class="font-medium">{tx.payee}</span>
              </p>
              <div class="mt-1 flex flex-wrap items-center gap-1">
                <CategoryBadge name={tx.category} type={tx.categoryType} />
                {#each tx.tags as tag (tag)}<TagChip {tag} cardKeywords={data.cardKeywords} />{/each}
                {#if rule}
                  <Badge status={MCC_STATUS[mcc.status].tone}
                    >{mcc.code
                      ? `MCC ${String(mcc.code).padStart(4, '0')} ${MCC_STATUS[mcc.status].label}`
                      : MCC_STATUS[mcc.status].label}</Badge
                  >
                  <span class="muted text-xs">{rule}</span>
                {/if}
              </div>
            </div>
            <div class="text-right">
              <Money value={(tx.deposit ?? 0) - (tx.payment ?? 0)} tone />
              {#if rule}<p class="muted text-xs">+<Money value={cashback} /> cashback</p>{/if}
            </div>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No transactions in the current period.</p>
    {/if}
  </Card>
{:else if activeTab === 'history'}
  <!-- WM-CARD-11 -->
  <Card title="Cashback reconciliation" subtitle="Each period's refund against the cashback the rules give">
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
        <p class="muted text-xs">Estimated cashback</p>
        <Money value={ytd.estimated} />
      </div>
      <div>
        <p class="muted text-xs">Refund</p>
        <Money value={ytd.refund} />
      </div>
      <div>
        <p class="muted text-xs">Missed</p>
        <Money value={ytd.missed} />
      </div>
    </div>
    <BarChart
      labels={yearHistory.map((m) => formatMonth(m.month, true))}
      series={barSeries}
      label="Refund vs missed vs estimated cashback"
    />
    {#if cap === null}
      <p class="muted mt-1 text-xs">No period cap: at least one rule is uncapped.</p>
    {/if}
    <h3 class="mt-6 mb-2 text-sm font-semibold">Cumulative {activeYear}</h3>
    <MultiLineChart
      dates={yearHistory.map((m) => m.month)}
      series={cumulativeSeries}
      format={formatMasked}
      height={200}
      label="Cumulative cashback year to date"
    />
    <div class="-mx-5 mt-4 overflow-x-auto">
      <table class="w-full text-sm">
        <thead
          ><tr class="muted text-left text-xs uppercase"
            ><th class="px-5 py-2">Period</th><th class="px-5 py-2 text-right">Spent</th><th
              class="px-5 py-2 text-right">Confirmed</th
            ><th class="px-5 py-2 text-right">Estimated</th><th class="px-5 py-2 text-right">Refund</th><th
              class="px-5 py-2">Status</th
            ></tr
          ></thead
        >
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each [...yearHistory].reverse() as m (m.month)}
            <tr>
              <td class="px-5 py-2"
                >{formatMonth(m.month)}
                {#if m.start && m.end}<span class="muted block text-xs"
                    >{formatDate(m.start)} - {formatDate(m.end)}</span
                  >{/if}</td
              >
              <td class="px-5 py-2 text-right"><Money value={m.spent} /></td>
              <td class="px-5 py-2 text-right"><Money value={m.cashback.confirmed} /></td>
              <td class="px-5 py-2 text-right"><Money value={m.cashback.total} /></td>
              <td class="px-5 py-2 text-right"><Money value={m.refund} /></td>
              <td class="px-5 py-2" title={RECONCILIATION[m.status].hint}
                ><Badge status={RECONCILIATION[m.status].tone}>{RECONCILIATION[m.status].label}</Badge></td
              >
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted mt-3 text-xs">
      A refund counts for the period starting in its effective month: write that month (YYYY-MM) in the refund's memo.
    </p>
  </Card>
{:else}
  <Card
    title="Cashback rules"
    subtitle="{card.versionFrom
      ? `Terms from ${formatDate(card.versionFrom)}, counted per ${periodLabel}. `
      : ''}A payment matches by MCC (an mcc:NNNN tag, else the Merchants tab). Exclusions win, then the highest tier the payment and period spend qualify for, then the fallback."
  >
    {#if card.cap > 0}<p class="mb-3 text-sm">Card cap <Money value={card.cap} /> per {periodLabel}.</p>{/if}
    <div class="overflow-x-auto">
      <table class="w-full text-sm">
        <thead
          ><tr class="muted text-left text-xs uppercase"
            ><th class="py-2">Rule</th><th class="py-2">MCCs</th><th class="py-2 text-right">Rate</th><th
              class="py-2 text-right">Min payment</th
            ><th class="py-2 text-right">Min period spend</th><th class="py-2 text-right">Cap</th></tr
          ></thead
        >
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each card.rules as rule (rule.name)}
            <tr>
              <td class="py-2"
                >{rule.name}
                {#if rule.fallback}<Badge>fallback</Badge>{/if}
                {#if rule.exclude}<Badge>no cashback</Badge>{/if}</td
              >
              <td class="py-2">
                {#if rule.mccs.length}<span class="tabular text-xs">{formatMccs(rule.mccs)}</span>{:else}<span
                    class="muted">{rule.fallback ? 'Other spend' : '-'}</span
                  >{/if}
              </td>
              {#if rule.exclude}
                <td class="muted py-2 text-right" colspan="4">Excluded</td>
              {:else}
                <td class="tabular py-2 text-right">{formatPercent(rule.rate, 1)}</td>
                <td class="py-2 text-right"
                  >{#if rule.minTransaction > 0}<Money value={rule.minTransaction} />{:else}<span class="muted">-</span
                    >{/if}</td
                >
                <td class="py-2 text-right"
                  >{#if rule.minCycleSpend > 0}<Money value={rule.minCycleSpend} />{:else}<span class="muted">-</span
                    >{/if}</td
                >
                <td class="py-2 text-right"
                  >{#if rule.cap > 0}<Money value={rule.cap} />{#if rule.capGroup}<span class="muted block text-xs"
                        >shared: {rule.capGroup}</span
                      >{/if}{:else}<span class="muted">No cap</span>{/if}</td
                >
              {/if}
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </Card>
{/if}

<!-- WM-CARD-5 -->
<div class="mt-4">
  <label class="label" for="summary-month">AI summary period</label>
  <select id="summary-month" class="field mb-3 w-auto" bind:value={summaryMonth}>
    {#each summaryMonths as month (month)}
      <option value={month}>{formatMonth(month)}</option>
    {/each}
  </select>
  <RecommendationsPanel
    title="Recommendations"
    subtitle="From the period summary of {card.name} · {formatMonth(summaryMonth)}"
    page="/accounts/credit-cards"
    review={(force, signal) =>
      api.review('credit-summary', { card: card.name, month: summaryMonth }, { force, signal })}
  />
</div>
