<script lang="ts">
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import type { Allocation, FundPerformance, PortfolioSection, PortfolioValuation } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import AllocationCard from '$lib/investments/AllocationCard.svelte';
  import { BANK_RATES_HREF, bestBankRate } from '$lib/investments/bankRates';
  import { settle } from '$lib/investments/fmarket';
  import FundPerformanceCard from '$lib/investments/FundPerformanceCard.svelte';
  import SectionError from '$lib/investments/fmarket/SectionError.svelte';
  import HoldingsCard from '$lib/investments/HoldingsCard.svelte';
  import { nf } from '$lib/investments/holdings';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { rateDetail } from '$lib/rates';
  import { STATUS } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();

  // A forced refresh or a target change replaces the loaded figures until the next navigation.
  let refreshed = $state<PortfolioValuation | null>(null);
  let updatedAllocation = $state<Allocation | null>(null);
  let refreshedFunds = $state<FundPerformance | null>(null);
  let isRefreshing = $state(false);
  const valuation = $derived(refreshed ?? data.valuation);
  const totals = $derived(valuation.totals);
  const allocation = $derived(updatedAllocation ?? (data.allocation.ok ? data.allocation.value : null));
  const fundPerformance = $derived(refreshedFunds ?? (data.fundPerformance.ok ? data.fundPerformance.value : null));
  // Valuation issues and ignored Preferences rows, each once.
  const issues = $derived([...new Set([...valuation.issues, ...(allocation?.issues ?? [])])]);

  async function refresh() {
    isRefreshing = true;
    try {
      // Fund performance settles on its own so a NAV outage does not block the prices.
      const [nextValuation, nextAllocation, nextFunds] = await Promise.all([
        api.valuation({ force: true }),
        api.allocation({ force: true }),
        settle(() => api.fundPerformance({ force: true })),
      ]);
      [refreshed, updatedAllocation] = [nextValuation, nextAllocation];
      if (nextFunds.ok) refreshedFunds = nextFunds.value;
      else toasts.error(new Error(nextFunds.error), 'Could not refresh fund performance');
    } catch (error) {
      toasts.error(error, 'Could not refresh prices');
    } finally {
      isRefreshing = false;
    }
  }

  const sectionTitles: Record<PortfolioSection, string> = { crypto: 'Crypto', funds: 'Fund certificates' };
  const sections = $derived(
    (['crypto', 'funds'] as const).map((key) => {
      const rows = valuation.holdings.filter((h) => h.section === key);
      return { key, title: sectionTitles[key], rows, total: rows.reduce((sum, h) => sum + (h.valueVND ?? 0), 0) };
    }),
  );

  // One bank-rates list (ROADMAP C5): the full list is on the Fmarket page.
  const bestRate = $derived(data.bankRates.ok ? bestBankRate(data.bankRates.value.rates) : null);
</script>

<PageHeader title="Investments" subtitle="Holdings from the sheet, valued in VND">
  {#snippet actions()}
    <button type="button" class="btn-ghost" disabled={isRefreshing} onclick={refresh}>
      <Icon name="refresh" size={16} class={isRefreshing ? 'animate-spin' : ''} /> Refresh prices
    </button>
  {/snippet}
</PageHeader>

{#if issues.length}
  <div class="glass mb-4 border p-3 text-sm {STATUS.warning.box}" role="status">
    <p class="flex items-center gap-2 font-medium">
      <Icon name={STATUS.warning.icon} size={16} />Some figures are incomplete:
    </p>
    <ul class="mt-1 list-disc pl-5">
      {#each issues as issue, i (i)}<li>{issue}</li>{/each}
    </ul>
  </div>
{/if}

<!-- WM-INV-4, WM-RULE-8 -->
<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Stat
    label="Portfolio (VND)"
    value={totals.total}
    hint={totals.unvalued ? `${totals.unvalued} items not valued` : undefined}
  />
  <Stat label="VND assets" value={totals.vnd} hint="Accounts {nf.format(totals.vndAccounts)} + funds" />
  <Stat
    label="Crypto"
    value={totals.crypto}
    hint="Max of accounts {nf.format(totals.cryptoAccounts)} and holdings {nf.format(totals.cryptoHoldings)}"
  />
  <Stat label="Exchange rates">
    {#if valuation.rates.length}
      {#each valuation.rates as rate (rate.from)}
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

<!-- WM-INV-20, ROADMAP C4: the one engine allocation with targets and drift -->
{#if allocation}
  <AllocationCard {allocation} onchange={(next) => (updatedAllocation = next)} />
{:else if !data.allocation.ok}
  <Card title="Allocation" class="mt-4">
    <SectionError title="The allocation is unavailable" message={data.allocation.error} />
  </Card>
{/if}

{#each sections as { key, title, rows, total } (key)}
  <HoldingsCard {title} {rows} {total} />
  {#if key === 'funds'}
    <!-- ROADMAP M3, WM-INV-21/22: fund holdings vs a bank deposit -->
    {#if fundPerformance}
      <FundPerformanceCard performance={fundPerformance} />
    {:else if !data.fundPerformance.ok}
      <Card title="Funds vs a bank deposit" class="mt-4">
        <SectionError title="Fund performance is unavailable" message={data.fundPerformance.error} />
      </Card>
    {/if}
  {/if}
{/each}

<p class="muted mt-2 text-right text-xs">Valued {formatDate(valuation.generatedAt)}</p>

<!-- ROADMAP C5: one bank-rates list, on the Fmarket page -->
<p class="glass mt-4 flex flex-wrap items-center gap-x-3 gap-y-1 px-5 py-3 text-sm">
  <span class="font-medium">Best savings rate</span>
  {#if !data.bankRates.ok}
    <span class="muted">unavailable right now: {data.bankRates.error}</span>
  {:else if bestRate}
    <span
      >{bestRate.bank}{bestRate.tenor ? ` · ${bestRate.tenor}` : ''}
      <span class="tabular font-semibold">{formatPercent(bestRate.rate, 2)}</span> per year</span
    >
    {#if data.bankRates.value.source}<SourceBadge source={data.bankRates.value.source} />{/if}
  {:else}
    <span class="muted">no bank published a rate</span>
  {/if}
  <a href="{base}{BANK_RATES_HREF}" class="ml-auto text-indigo-600 hover:underline dark:text-indigo-400"
    >All bank rates</a
  >
</p>
