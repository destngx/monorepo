<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { PortfolioSection, PortfolioValuation } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import {
    ASSET_CLASSES,
    classValues,
    computeDrift,
    DRIFT_THRESHOLD,
    parseTargets,
    serializeTargets,
    targetsError,
    type Targets,
  } from '$lib/investments/allocation';
  import HoldingsCard from '$lib/investments/HoldingsCard.svelte';
  import { nf } from '$lib/investments/holdings';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { rateDetail } from '$lib/rates';
  import { targetAllocation } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();

  // A forced refresh replaces the loaded valuation until the next navigation.
  let refreshed = $state<PortfolioValuation | null>(null);
  let isRefreshing = $state(false);
  const valuation = $derived(refreshed ?? data.valuation);
  const totals = $derived(valuation.totals);

  async function refresh() {
    isRefreshing = true;
    try {
      refreshed = await api.valuation({ force: true });
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

  // Target allocation and drift (WM-INV-20); targets live in this browser only.
  const targets = $derived(parseTargets(targetAllocation.value));
  const drift = $derived(targets ? computeDrift(classValues(totals), targets) : null);
  const breaches = $derived(drift?.rows.filter((row) => row.breach) ?? []);
  let editingTargets = $state(false);
  let draft = $state<Targets>({ cash: 0, funds: 0, crypto: 0 });

  function editTargets() {
    draft = targets ? { ...targets } : { cash: 0, funds: 0, crypto: 0 };
    editingTargets = true;
  }

  function saveTargets(event: SubmitEvent) {
    event.preventDefault();
    const next = { cash: Number(draft.cash), funds: Number(draft.funds), crypto: Number(draft.crypto) };
    const problem = targetsError(next);
    if (problem) {
      toasts.push('warning', problem);
      return;
    }
    targetAllocation.value = serializeTargets(next);
    editingTargets = false;
    toasts.push('success', 'Target allocation saved.');
  }

  function clearTargets() {
    targetAllocation.value = '';
    editingTargets = false;
  }

  const bankRates = $derived(
    data.bankRates.ok ? [...data.bankRates.value.rates].sort((a, b) => b.rate - a.rate).slice(0, 15) : [],
  );
</script>

<PageHeader title="Investments" subtitle="Holdings from the sheet, valued in VND by the engine">
  {#snippet actions()}
    <button type="button" class="btn-ghost" disabled={isRefreshing} onclick={refresh}>
      <Icon name="refresh" size={16} class={isRefreshing ? 'animate-spin' : ''} /> Refresh prices
    </button>
  {/snippet}
</PageHeader>

{#if valuation.issues.length}
  <div class="glass mb-4 border border-amber-500/40 p-3 text-sm text-amber-800 dark:text-amber-200" role="status">
    <p class="font-medium">Some figures are incomplete:</p>
    <ul class="mt-1 list-disc pl-5">
      {#each valuation.issues as issue, i (i)}<li>{issue}</li>{/each}
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

<!-- WM-INV-20 -->
<Card
  title="Target allocation"
  subtitle="Rebalance when a class drifts more than {DRIFT_THRESHOLD} points from its target"
  class="mt-4"
>
  {#snippet actions()}
    {#if !editingTargets}
      <button type="button" class="btn-ghost text-xs" onclick={editTargets}
        ><Icon name="pencil" size={14} /> {targets ? 'Edit targets' : 'Set targets'}</button
      >
    {/if}
  {/snippet}
  {#if editingTargets}
    <form class="grid gap-3 sm:grid-cols-4 sm:items-end" onsubmit={saveTargets}>
      {#each ASSET_CLASSES as assetClass (assetClass.key)}
        <div>
          <label class="label" for="target-{assetClass.key}">{assetClass.label} (%)</label>
          <input
            id="target-{assetClass.key}"
            type="number"
            min="0"
            max="100"
            step="0.5"
            class="field"
            bind:value={draft[assetClass.key]}
          />
        </div>
      {/each}
      <div class="flex gap-2">
        <button type="submit" class="btn-primary">Save</button>
        <button type="button" class="btn-ghost" onclick={() => (editingTargets = false)}>Cancel</button>
        {#if targets}<button type="button" class="btn-ghost" onclick={clearTargets}>Clear</button>{/if}
      </div>
    </form>
  {:else if drift}
    {#if drift.shouldRebalance}
      <div
        class="mb-3 rounded-xl border border-orange-500/40 p-3 text-sm text-orange-800 dark:text-orange-200"
        role="alert"
      >
        Rebalance suggested: {breaches
          .map(
            (row) =>
              `${row.label} is ${row.drift > 0 ? 'over' : 'under'} target by ${Math.abs(row.drift).toFixed(1)} points`,
          )
          .join('; ')}.
      </div>
    {/if}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[36rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Class</th>
            <th class="px-5 py-2 text-right">Value</th>
            <th class="px-5 py-2 text-right">Current</th>
            <th class="px-5 py-2 text-right">Target</th>
            <th class="px-5 py-2 text-right">Drift</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each drift.rows as row (row.key)}
            <tr>
              <td class="px-5 py-2">{row.label}</td>
              <td class="px-5 py-2 text-right"><Money value={row.value} /></td>
              <td class="tabular px-5 py-2 text-right">{formatPercent(row.currentPercent)}</td>
              <td class="tabular px-5 py-2 text-right">{formatPercent(row.targetPercent)}</td>
              <td class="px-5 py-2 text-right">
                <span class="tabular {row.breach ? 'font-semibold text-orange-600 dark:text-orange-400' : ''}"
                  >{row.drift > 0 ? '+' : ''}{row.drift.toFixed(1)} pts</span
                >
                {#if row.breach}<Badge tone="orange">Rebalance</Badge>{/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if totals.unvalued}
      <p class="muted mt-2 text-xs">{totals.unvalued} items are not valued and are not counted in the drift.</p>
    {/if}
  {:else}
    <p class="muted text-sm">
      Set a target share for cash, fund certificates and crypto to track drift. Targets are stored in this browser.
    </p>
  {/if}
</Card>

{#each sections as { key, title, rows, total } (key)}
  <HoldingsCard {title} {rows} {total} />
{/each}

<p class="muted mt-2 text-right text-xs">Valued {formatDate(valuation.generatedAt)}</p>

<Card title="Bank savings rates" subtitle="Highest published rates" class="mt-4">
  {#snippet actions()}
    {#if data.bankRates.ok && data.bankRates.value.source}<SourceBadge source={data.bankRates.value.source} />{/if}
  {/snippet}
  {#if !data.bankRates.ok}
    <p class="muted text-sm">Bank rates are unavailable right now: {data.bankRates.error}</p>
  {:else if bankRates.length}
    <ul class="grid gap-x-6 gap-y-1 text-sm sm:grid-cols-2 lg:grid-cols-3">
      {#each bankRates as rate, i (i)}
        <li class="flex justify-between gap-3 border-b border-slate-900/5 py-1.5 dark:border-white/5">
          <span>{rate.bank}{rate.tenor ? ` · ${rate.tenor}` : ''}</span>
          <span class="tabular font-medium">{formatPercent(rate.rate, 2)}</span>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted text-sm">No bank published a rate.</p>
  {/if}
</Card>
