<script lang="ts">
  import { base } from '$app/paths';
  import { track, Button, Deferred, Icon, SkeletonCard, SkeletonStat } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import { defer, settled, type Streamed } from '$lib/api/load';
  import type {
    Allocation,
    Exposure,
    FundPerformance,
    Hedges,
    PortfolioSection,
    PortfolioValuation,
  } from '$lib/api/types';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import AllocationCard from '$lib/investments/AllocationCard.svelte';
  import { BANK_RATES_HREF, bestBankRate } from '$lib/investments/bankRates';
  import CryptoCard from '$lib/investments/CryptoCard.svelte';
  import CryptoHoldingsModal from '$lib/investments/CryptoHoldingsModal.svelte';
  import ExposureCard from '$lib/investments/ExposureCard.svelte';
  import FundPerformanceCard from '$lib/investments/FundPerformanceCard.svelte';
  import FundTradeModal from '$lib/investments/FundTradeModal.svelte';
  import HedgesCard from '$lib/investments/HedgesCard.svelte';
  import HoldingsCard from '$lib/investments/HoldingsCard.svelte';
  import { nf } from '$lib/investments/holdings';
  import { rateDetail } from '$lib/rates';
  import { STATUS } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();

  // Each section streams in from the page data. A forced refresh or a saved
  // target swaps in new promises until the next load: the sections keep their
  // figures with a Refreshing badge meanwhile, and show their own error if
  // the refresh fails.
  type Parts = {
    valuation: Streamed<PortfolioValuation>;
    allocation: Streamed<Allocation>;
    fundPerformance: Streamed<FundPerformance>;
    exposure: Streamed<Exposure>;
    hedges: Streamed<Hedges>;
  };
  let forced = $state.raw<{ from: typeof data; parts: Partial<Parts> } | null>(null);
  const parts = $derived<Parts>({ ...data, ...(forced?.from === data ? forced.parts : {}) });
  function replace(next: Partial<Parts>) {
    forced = { from: data, parts: { ...(forced?.from === data ? forced.parts : {}), ...next } };
  }

  const valuation = track(() => parts.valuation);
  const allocation = track(() => parts.allocation);
  const exposure = track(() => parts.exposure);
  const bankRates = track(() => data.bankRates);
  // Valuation issues, ignored Preferences rows and missing price moves, each once.
  const issues = $derived([
    ...new Set([
      ...(valuation.value?.issues ?? []),
      ...(allocation.value?.issues ?? []),
      ...(exposure.value?.issues ?? []),
    ]),
  ]);

  async function refresh() {
    const next: Parts = {
      valuation: defer(() => api.valuation({ force: true })),
      allocation: defer(() => api.allocation({ force: true })),
      fundPerformance: defer(() => api.fundPerformance({ force: true })),
      exposure: defer(() => api.exposure({ force: true })),
      hedges: defer(() => api.hedges({ force: true })),
    };
    replace(next);
    const results = await Promise.all(Object.values(next));
    const failed = results.filter((result) => !result.ok).length;
    if (failed) toasts.push('warning', `${failed} of ${results.length} sections could not refresh; each one says why.`);
    else toasts.success('Prices refreshed.');
  }

  const sectionTitles: Record<PortfolioSection, string> = { crypto: 'Crypto', funds: 'Fund certificates' };
  function sectionsOf(value: PortfolioValuation) {
    return (['crypto', 'funds'] as const).map((key) => {
      const rows = value.holdings.filter((h) => h.section === key);
      return { key, title: sectionTitles[key], rows, total: rows.reduce((sum, h) => sum + (h.valueVND ?? 0), 0) };
    });
  }

  // Writes for what was edited in the sheet only (WM-FILL-10, WM-FILL-11).
  let writing = $state<PortfolioSection | null>(null);
  const heldIn = (section: PortfolioSection) => valuation.value?.holdings.filter((h) => h.section === section) ?? [];

  // One bank-rates list (ROADMAP C5): the full list is on the Fmarket page.
  const bestRate = $derived(bankRates.value ? bestBankRate(bankRates.value.rates) : null);
</script>

<PageHeader title="Investments" subtitle="What you hold and how it moves with markets">
  {#snippet actions()}
    <Button icon="refresh" busyText="Refreshing…" errorContext="Could not refresh prices" onclick={refresh}
      >Refresh prices</Button
    >
  {/snippet}
</PageHeader>

{#if issues.length}
  <div class="panel mb-4 border p-3 text-sm {STATUS.warning.box}" role="status">
    <p class="flex items-center gap-2 font-medium">
      <Icon name={STATUS.warning.icon} size={16} />Some figures are incomplete:
    </p>
    <ul class="mt-1 list-disc pl-5">
      {#each issues as issue, i (i)}<li>{issue}</li>{/each}
    </ul>
  </div>
{/if}

<!-- ROADMAP M1: the headline answer first, from the one allocation -->
<Deferred data={parts.exposure} label="your exposure">
  {#snippet children(value)}<ExposureCard exposure={value} />{/snippet}
  {#snippet skeleton()}<SkeletonCard title="My exposure" lines={3} chart={120} />{/snippet}
</Deferred>

<!-- ROADMAP M2 and M4: the crypto view, then gold and USD as protection -->
<div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
  <!-- Crypto comes from the exposure: when that fails, its error shows once, in the exposure card above. -->
  {#if exposure.value || !exposure.failure}
    <Deferred data={parts.exposure} label="crypto">
      {#snippet children(value)}<CryptoCard crypto={value.crypto} netWorth={value.netWorth} />{/snippet}
      {#snippet skeleton()}<SkeletonCard title="Crypto" lines={4} />{/snippet}
    </Deferred>
  {/if}
  <Deferred data={parts.hedges} label="gold and USD trends">
    {#snippet children(value)}<HedgesCard hedges={value} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Gold and USD as protection" lines={2} chart={120} />{/snippet}
  </Deferred>
</div>

<!-- WM-INV-4, WM-RULE-8 -->
<Deferred data={parts.valuation} label="portfolio totals" class="mt-4">
  {#snippet children(value)}
    {@const totals = value.totals}
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <Stat
        label="Portfolio (VND)"
        value={totals.total}
        hint={totals.unvalued ? `${totals.unvalued} items not valued` : undefined}
      />
      <!-- The hints hold amounts, so they go through Money and hide with the privacy mask (WM-UI-2). -->
      <Stat label="VND assets">
        <Money value={totals.vnd} />
        <span class="muted mt-1 block text-xs font-normal" data-testid="vnd-hint"
          >Accounts <Money value={totals.vndAccounts} /> + funds</span
        >
      </Stat>
      <Stat label="Crypto">
        <Money value={totals.crypto} />
        <span class="muted mt-1 block text-xs font-normal" data-testid="crypto-hint"
          >Max of accounts <Money value={totals.cryptoAccounts} /> and holdings <Money
            value={totals.cryptoHoldings}
          /></span
        >
      </Stat>
      <Stat label="Exchange rates">
        {#if value.rates.length}
          {#each value.rates as rate (rate.from)}
            <span class="block text-base">
              <span class="muted text-xs font-normal">{rate.from}</span>
              <span class="tabular">{nf.format(rate.rate)}</span>
            </span>
            <span class="block text-xs font-normal"
              ><SourceBadge source={rate.source ?? { provider: rate.provider }} detail={rateDetail(rate)} /></span
            >
          {/each}
        {:else}<span class="muted">-</span>{/if}
      </Stat>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      {#each ['portfolio total', 'VND assets', 'crypto total', 'exchange rates'] as label (label)}
        <SkeletonStat {label} />
      {/each}
    </div>
  {/snippet}
</Deferred>

<!-- WM-INV-20, ROADMAP C4: the one engine allocation with targets and drift -->
<Deferred data={parts.allocation} label="the allocation" class="mt-4">
  {#snippet children(value)}
    <AllocationCard allocation={value} onchange={(next) => replace({ allocation: settled(next) })} />
  {/snippet}
  {#snippet skeleton()}<SkeletonCard title="Allocation" lines={2} rows={4} />{/snippet}
</Deferred>

<Deferred data={parts.valuation} label="holdings" class="mt-4">
  {#snippet children(value)}
    <div class="space-y-4">
      {#each sectionsOf(value) as { key, title, rows, total } (key)}
        <HoldingsCard {title} {rows} {total}>
          {#snippet action()}
            <Button size="sm" icon={key === 'funds' ? 'plus' : 'pencil'} onclick={() => (writing = key)}
              >{key === 'funds' ? 'Record a trade' : 'Update balances'}</Button
            >
          {/snippet}
        </HoldingsCard>
      {/each}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div class="space-y-4">
      <SkeletonCard title="Crypto" lines={1} rows={4} />
      <SkeletonCard title="Fund certificates" lines={1} rows={4} />
    </div>
  {/snippet}
</Deferred>

<!-- ROADMAP M3, WM-INV-21/22: fund holdings vs a bank deposit -->
<Deferred data={parts.fundPerformance} label="fund performance" class="mt-4">
  {#snippet children(value)}<FundPerformanceCard performance={value} />{/snippet}
  {#snippet skeleton()}<SkeletonCard title="Funds vs a bank deposit" lines={3} chart={160} />{/snippet}
</Deferred>

{#if valuation.value}
  <p class="muted mt-2 text-right text-xs">Valued {formatDate(valuation.value.generatedAt)}</p>
{/if}

<!-- ROADMAP C5: one bank-rates list, on the Fmarket page -->
<p class="panel mt-4 flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 text-sm" aria-busy={bankRates.pending}>
  <span class="font-medium">Best savings rate</span>
  {#if bankRates.loading}
    <span class="skeleton inline-block h-3.5 w-48" role="status"><span class="sr-only">Loading bank rates…</span></span>
  {:else if bankRates.failure && !bankRates.value}
    <span class="muted">unavailable right now: {bankRates.failure.message}</span>
  {:else if bestRate}
    <span
      >{bestRate.bank}{bestRate.tenor ? ` · ${bestRate.tenor}` : ''}
      <span class="tabular font-semibold">{formatPercent(bestRate.rate, 2)}</span> per year</span
    >
    {#if bankRates.value?.source}<SourceBadge source={bankRates.value.source} />{/if}
  {:else}
    <span class="muted">no bank published a rate</span>
  {/if}
  <a href="{base}{BANK_RATES_HREF}" class="link ml-auto">All bank rates</a>
</p>

<FundTradeModal
  open={writing === 'funds'}
  codes={[
    ...new Set(
      heldIn('funds')
        .map((h) => h.symbol)
        .filter(Boolean),
    ),
  ]}
  onclose={() => (writing = null)}
/>
<CryptoHoldingsModal open={writing === 'crypto'} holdings={heldIn('crypto')} onclose={() => (writing = null)} />
