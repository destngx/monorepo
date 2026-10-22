<script lang="ts">
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { track, Card, Deferred, EmptyState, SkeletonCard, SkeletonStat } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import type { CardBankReport } from '$lib/api/types';
  import { base } from '$app/paths';
  import BestCardForm from '$lib/cards/BestCardForm.svelte';
  import {
    allCards,
    capsLeft,
    compareCards,
    historyYears,
    portfolioTotals,
    UTILIZATION,
    visibleCards,
    type CardMetric,
  } from '$lib/cards/cards';
  import ProgramComparison from '$lib/cards/ProgramComparison.svelte';
  import { PALETTE } from '$lib/charts/palette';
  import Badge from '$lib/components/Badge.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import MultiLineChart from '$lib/investments/fmarket/MultiLineChart.svelte';
  import { formatCompact, formatDate, formatMoney, formatPercent } from '$lib/format';
  import Masked from '$lib/components/Masked.svelte';
  import { STATUS } from '$lib/status';

  // The header renders at once; the banks and their cards stream in, with
  // the best-card form and programme comparison once there are cards.
  let { data } = $props();
  const report = track(() => data.banks);
  const banks = $derived<CardBankReport[]>(report.value ?? []);
  const totals = $derived(portfolioTotals(banks));
  const utilization = $derived(totals.limit ? (totals.usage / totals.limit) * 100 : 0);

  // Closed cards are hidden unless asked for; totals still count them.
  let showClosed = $state(false);
  const closedCount = $derived(allCards(banks).filter((card) => card.closedFrom).length);

  // Cross-card history comparison (WM-CARD-2).
  const cards = $derived(visibleCards(allCards(banks), showClosed));
  const years = $derived(historyYears(cards));
  let year = $state<string | null>(null);
  const activeYear = $derived(year ?? years[0] ?? '');
  let metric = $state<CardMetric>('refund');
  const metrics: [CardMetric, string][] = [
    ['refund', 'Refund'],
    ['cashback', 'Estimated cashback'],
    ['spent', 'Spent'],
    ['efficiency', 'Efficiency %'],
  ];
  // Cards hidden from the comparison; every card is shown by default.
  let hidden = $state<Record<string, boolean>>({});
  const comparison = $derived(
    compareCards(
      cards.filter((card) => !hidden[card.name]),
      activeYear,
      metric,
    ),
  );
  const colorOf = (name: string) => PALETTE[cards.findIndex((card) => card.name === name) % PALETTE.length];
  const formatMetric = (value: number) => (metric === 'efficiency' ? formatPercent(value, 0) : formatCompact(value));
  // Amounts inside a card link: the pointer-only mask token, never a nested button (WM-UI-15).
  const toneOf = (value: number) =>
    value > 0 ? 'text-emerald-600 dark:text-emerald-400' : value < 0 ? 'text-orange-600 dark:text-orange-400' : '';
</script>

<PageHeader title="Credit Cards" subtitle="Cashback, bank limits and pay-by dates">
  {#snippet actions()}
    {#if closedCount}
      <label class="flex items-center gap-2 text-sm">
        <input type="checkbox" bind:checked={showClosed} />
        Show closed cards ({closedCount})
      </label>
    {/if}
  {/snippet}
</PageHeader>

<Deferred data={data.banks} label="credit cards">
  {#snippet children()}
    <div>
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
        <Stat label="Total limit" value={totals.limit} />
        <Stat label="Used" value={totals.usage} hint=" {formatPercent(utilization, 0)} utilisation" />
        <Stat label="Cashback this period" value={totals.cashback.total}>
          {#snippet hint()}<Masked text={formatCompact(totals.cashback.confirmed)} /> confirmed{/snippet}
        </Stat>
        <Stat label="Refunds YTD" value={totals.refundYtd} />
        <!-- Every card, closed ones included: fees count once paid (memo "bankfee"). -->
        <Stat label="Fees paid YTD">
          <Money value={totals.feesPaidYtd} />
          <ul class="mt-1.5 space-y-0.5 text-xs font-normal">
            {#each allCards(banks) as card (card.name)}
              <li class="flex justify-between gap-2">
                <span class="muted truncate" title={card.name}>{card.name}{card.closedFrom ? ' (closed)' : ''}</span>
                {#if card.feesPaidYtd === 0 && !card.closedFrom && card.annualFee > 0}
                  <span class="muted whitespace-nowrap">unpaid · <Money value={card.annualFee} /></span>
                {:else}
                  <Money value={card.feesPaidYtd} />
                {/if}
              </li>
            {/each}
          </ul>
        </Stat>
        <Stat label="Net earn YTD"><Money value={totals.netEarnYtd} tone signed /></Stat>
      </div>

      {#if banks.length === 0}
        <Card class="mt-4"
          ><EmptyState
            title="No credit cards configured"
            detail="Add the Programs, ProgramRules, Banks and Cards tabs (and Merchants for payee MCCs) to the Google Sheet. See the setup guide for their columns."
          /></Card
        >
      {/if}

      {#each banks as bank (bank.name)}
        {@const bankCards = visibleCards(bank.cards, showClosed)}
        {@const hiddenCount = bank.cards.length - bankCards.length}
        {@const use = UTILIZATION[bank.utilizationBand]}
        <!-- WM-CARD-16: the limit and usage are the bank's, shared by its cards -->
        <Card
          title={bank.name}
          subtitle="{bank.sharedAccount ? `${bank.sharedAccount} · ` : ''}{bankCards.length} {bankCards.length === 1
            ? 'card'
            : 'cards'}{hiddenCount ? ` · ${hiddenCount} closed hidden` : ''}"
          class="mt-6"
        >
          {#snippet actions()}
            <span class="muted text-xs"
              >Limit <Money value={bank.limit} /> · Room <Money value={bank.remainingLimit} /></span
            >
          {/snippet}
          <div class="mb-4 flex items-center gap-2">
            <ProgressBar percent={bank.utilization} color={STATUS[use.tone].fill} label="Bank utilisation" />
            <span class="muted text-xs whitespace-nowrap">{formatPercent(bank.utilization, 0)} used</span>
            <Badge status={use.tone}>{use.label}</Badge>
          </div>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            {#each bankCards as card (card.name)}
              {@const standing = card.standing}
              <a
                href="{base}/accounts/credit-cards/{encodeURIComponent(card.name)}"
                class="interactive rounded-xl border border-stone-900/10 p-4 hover:bg-accent-500/5 dark:border-white/10"
              >
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="font-medium">{card.name}</p>
                    <p class="muted text-xs">
                      {#if card.closedFrom}
                        {card.program} · closed {formatDate(card.closedFrom)}
                      {:else}
                        {card.program} · pay by {formatDate(card.cycle.payBy)} · {card.cycle.daysUntilPayBy} days
                      {/if}
                    </p>
                  </div>
                  {#if card.closedFrom}<Badge>Closed</Badge>{:else if card.cycle.payByUrgent}<Badge status="critical"
                      >Pay soon</Badge
                    >{/if}
                </div>
                <!-- WM-CARD-1, WM-CARD-12 -->
                <dl class="mt-3 grid grid-cols-3 gap-2 text-xs">
                  {#if !card.closedFrom}
                    <div>
                      <dt class="muted">Spent this period</dt>
                      <dd><Masked inLink text={formatMoney(standing.spent)} /></dd>
                    </div>
                    <div>
                      <dt class="muted">Cashback</dt>
                      <dd><Masked inLink text={formatMoney(standing.cashback.total)} /></dd>
                    </div>
                    <div>
                      <dt class="muted">Confirmed</dt>
                      <dd><Masked inLink text={formatMoney(standing.cashback.confirmed)} /></dd>
                    </div>
                    <div>
                      <dt class="muted">Cap left</dt>
                      <dd>
                        {#each capsLeft(standing) as capped (capped.name)}
                          <p title={capped.name}><Masked inLink text={formatMoney(capped.left)} /></p>
                        {:else}<span class="muted">No cap</span>{/each}
                      </dd>
                    </div>
                  {/if}
                  <div>
                    <dt class="muted">Refunds YTD</dt>
                    <dd><Masked inLink text={formatMoney(card.refundYtd)} /></dd>
                  </div>
                  <div>
                    <dt class="muted">Net earn YTD</dt>
                    <dd><Masked inLink text={formatMoney(card.netEarnYtd)} class={toneOf(card.netEarnYtd)} /></dd>
                  </div>
                </dl>
                {#if standing.nextTier}
                  <p class="muted mt-2 text-xs">
                    <Masked inLink text={formatMoney(standing.nextTier.spendNeeded)} /> more this period reaches {standing.nextTier.rules.join(
                      ', ',
                    )}.
                  </p>
                {/if}
              </a>
            {/each}
          </div>
        </Card>
      {/each}

      {#if cards.length}
        <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
          <BestCardForm />
          <ProgramComparison />
        </div>
      {/if}

      {#if cards.length}
        <!-- WM-CARD-2: cross-card comparison -->
        <Card title="Compare cards" subtitle="History per card, by the month each period starts in" class="mt-4">
          {#snippet actions()}
            <select class="field w-auto py-1" aria-label="Metric" bind:value={metric}>
              {#each metrics as [value, text] (value)}<option {value}>{text}</option>{/each}
            </select>
            <select
              class="field w-auto py-1"
              aria-label="Year"
              value={activeYear}
              onchange={(e) => (year = e.currentTarget.value)}
            >
              {#each years as y (y)}<option value={y}>{y}</option>{/each}
            </select>
          {/snippet}
          <div class="mb-3 flex flex-wrap gap-2" role="group" aria-label="Cards to compare">
            {#each cards as card (card.name)}
              <button
                type="button"
                aria-pressed={!hidden[card.name]}
                title={hidden[card.name] ? `Show ${card.name}` : `Hide ${card.name}`}
                class="inline-flex items-center gap-1.5 rounded-md border px-2.5 py-1 text-xs hover:bg-stone-900/[0.05] active:bg-stone-900/10 dark:hover:bg-white/10 {hidden[
                  card.name
                ]
                  ? 'muted border-stone-900/10 dark:border-white/10'
                  : 'border-accent-400'}"
                onclick={() => (hidden[card.name] = !hidden[card.name])}
                ><span class="size-2.5 rounded-sm" style:background={colorOf(card.name)}></span>{card.name}</button
              >
            {/each}
          </div>
          <MultiLineChart
            dates={comparison.months}
            series={comparison.series.map((s) => ({ ...s, color: colorOf(s.name) }))}
            format={formatMetric}
            masked={metric !== 'efficiency'}
            label="Card comparison"
          />
        </Card>
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-3 xl:grid-cols-6">
        {#each { length: 6 }, index (index)}<SkeletonStat label="credit cards" />{/each}
      </div>
      <SkeletonCard lines={1} rows={2} label="credit cards" class="mt-6" />
    </div>
  {/snippet}
</Deferred>

<!-- WM-AI-5 -->
<RecommendationsPanel
  title="Recommendations"
  subtitle="From the credit card summary"
  page="/accounts/credit-cards"
  class="mt-4"
  review={(force, signal) => api.review('credit-summary', {}, { force, signal })}
/>
