<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { track, type Tracked, Card, Deferred, SkeletonChart, SkeletonTable, Spinner } from '@libs/ui-kit';
  import Badge from '$lib/components/Badge.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { rateLabel } from '$lib/bills/idleCash';
  import { STATUS } from '$lib/status';
  import { FUND_TYPES, type FundSummary, type FundType, type Sourced } from '$lib/investments/fmarket';
  import BankRatesChart from '$lib/investments/fmarket/BankRatesChart.svelte';
  import FundTable from '$lib/investments/fmarket/FundTable.svelte';
  import GoldPanel from '$lib/investments/fmarket/GoldPanel.svelte';

  let { data } = $props();

  type Tab = 'funds' | 'bank' | 'gold';
  const tabs: { key: Tab; label: string }[] = [
    { key: 'funds', label: 'Funds' },
    { key: 'bank', label: 'Bank rates' },
    { key: 'gold', label: 'Gold & USD' },
  ];

  // Tab and fund type live in the URL so back from a fund detail returns here.
  const tab = $derived<Tab>(tabs.find((t) => t.key === page.url.searchParams.get('tab'))?.key ?? 'funds');
  const fundType = $derived<FundType>(
    FUND_TYPES.find((t) => t.key === page.url.searchParams.get('type'))?.key ?? 'STOCK',
  );
  const fundLabel = $derived(FUND_TYPES.find((t) => t.key === fundType)?.label ?? '');
  // Each list streams in on its own (WM-UI-16); the filters show their counts as they arrive.
  const lists = {} as Record<FundType, Tracked<Sourced<FundSummary[]>>>;
  for (const { key } of FUND_TYPES) lists[key] = track(() => data.funds[key]);
  const funds = $derived(lists[fundType]);
  const bankRates = track(() => data.bankRates);
  const flexibleSavings = track(() => data.flexibleSavings);

  function select(key: 'tab' | 'type', value: string, fallback: string) {
    const url = new URL(page.url);
    if (value === fallback) url.searchParams.delete(key);
    else url.searchParams.set(key, value);
    void goto(url, { replaceState: true, keepFocus: true, noScroll: true });
  }

  const pill = (active: boolean) =>
    active
      ? 'bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900'
      : 'muted hover:bg-white/60 hover:text-stone-900 dark:hover:bg-white/10 dark:hover:text-white';
</script>

<PageHeader title="Fmarket" subtitle="Open-ended funds, bank savings rates, gold and USD from fmarket.vn" />

<!-- WM-INV-10 -->
<div
  class="mb-4 flex gap-1 overflow-x-auto rounded-2xl bg-stone-900/5 p-1 sm:inline-flex dark:bg-white/5"
  role="tablist"
>
  {#each tabs as t (t.key)}
    <button
      type="button"
      role="tab"
      aria-selected={tab === t.key}
      class="rounded-xl px-4 py-1.5 text-sm font-medium whitespace-nowrap transition active:scale-[0.97] {pill(
        tab === t.key,
      )}"
      onclick={() => select('tab', t.key, 'funds')}>{t.label}</button
    >
  {/each}
</div>

{#if tab === 'funds'}
  <Card title="{fundLabel} funds" subtitle="Ranked by 12-month return; select a fund for details">
    {#snippet actions()}
      {#if funds.value?.source}<SourceBadge source={funds.value.source} />{/if}
      <div
        class="flex flex-wrap gap-0.5 rounded-xl bg-stone-900/5 p-0.5 dark:bg-white/5"
        role="group"
        aria-label="Fund type"
      >
        {#each FUND_TYPES as t (t.key)}
          {@const list = lists[t.key]}
          <button
            type="button"
            aria-pressed={fundType === t.key}
            aria-busy={list.pending || undefined}
            class="inline-flex items-center rounded-lg px-2.5 py-1 text-xs font-medium transition active:scale-95 {pill(
              fundType === t.key,
            )}"
            onclick={() => select('type', t.key, 'STOCK')}
            >{t.label}{#if list.value}<span class="ml-1 opacity-60">{list.value.data.length}</span
              >{:else if list.loading}<Spinner size={10} class="ml-1 opacity-60" />{/if}</button
          >
        {/each}
      </div>
    {/snippet}
    <Deferred data={data.funds[fundType]} scope={fundType} label="{fundLabel.toLowerCase()} funds">
      {#snippet children(list)}<FundTable funds={list.data} label={fundLabel} />{/snippet}
      {#snippet skeleton()}<SkeletonTable rows={8} columns={5} label="{fundLabel.toLowerCase()} funds" />{/snippet}
    </Deferred>
  </Card>
{:else if tab === 'bank'}
  <Card title="Bank savings rates" subtitle="Highest published deposit rate per bank, % per year">
    {#snippet actions()}
      {#if bankRates.value?.source}<SourceBadge source={bankRates.value.source} />{/if}
    {/snippet}
    <Deferred data={data.bankRates} label="bank rates">
      {#snippet children(rates)}<BankRatesChart rates={rates.rates} />{/snippet}
      {#snippet skeleton()}<SkeletonChart height={260} label="bank rates" />{/snippet}
    </Deferred>
  </Card>
  <Card
    title="Wallet earning balances"
    subtitle="Live rates of e-wallet balances you can spend or withdraw any time, % per year"
    class="mt-4"
  >
    {#snippet actions()}
      {#if flexibleSavings.value?.source}<SourceBadge source={flexibleSavings.value.source} />{/if}
    {/snippet}
    <Deferred data={data.flexibleSavings} label="wallet earning balances">
      {#snippet children(savings)}
        <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5" data-testid="flexible-savings">
          {#each savings.products as product (product.id)}
            <li class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 py-1.5">
              <span class="min-w-0">
                {product.provider}
                {product.product}
                {#if product.url}
                  <a class="link text-xs" href={product.url} target="_blank" rel="noopener noreferrer"
                    >Open<span class="sr-only"> {product.provider} {product.product} (new tab)</span></a
                  >
                {/if}
              </span>
              <span class="flex items-center gap-1.5">
                {#if product.backing === 'business_cooperation'}<Badge status="warning">not a bank deposit</Badge>{/if}
                {#if product.stale}<Badge status="warning">stale</Badge>{/if}
                <span class="tabular font-medium">{rateLabel(product.rate)}</span>
              </span>
            </li>
          {:else}
            <li class="muted py-1.5">No wallet earning balance rate is published right now.</li>
          {/each}
        </ul>
        {#if savings.failed.length}
          <p class="mt-1 text-xs {STATUS.warning.ink}">Could not load: {savings.failed.join(', ')}.</p>
        {/if}
      {/snippet}
      {#snippet skeleton()}<SkeletonTable rows={2} columns={2} label="wallet earning balances" />{/snippet}
    </Deferred>
  </Card>
{:else}
  <GoldPanel />
{/if}
