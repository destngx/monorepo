<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import Card from '$lib/components/Card.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { FUND_TYPES, type FundType } from '$lib/investments/fmarket';
  import BankRatesChart from '$lib/investments/fmarket/BankRatesChart.svelte';
  import FundTable from '$lib/investments/fmarket/FundTable.svelte';
  import GoldPanel from '$lib/investments/fmarket/GoldPanel.svelte';
  import SectionError from '$lib/investments/fmarket/SectionError.svelte';

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
  const funds = $derived(data.funds[fundType]);

  function select(key: 'tab' | 'type', value: string, fallback: string) {
    const url = new URL(page.url);
    if (value === fallback) url.searchParams.delete(key);
    else url.searchParams.set(key, value);
    void goto(url, { replaceState: true, keepFocus: true, noScroll: true });
  }

  const pill = (active: boolean) =>
    active
      ? 'bg-white text-indigo-600 shadow-sm dark:bg-slate-800 dark:text-indigo-300'
      : 'muted hover:text-slate-900 dark:hover:text-white';
</script>

<PageHeader title="Fmarket" subtitle="Open-ended funds, bank savings rates, gold and USD from fmarket.vn" />

<!-- WM-INV-10 -->
<div
  class="mb-4 flex gap-1 overflow-x-auto rounded-2xl bg-slate-900/5 p-1 sm:inline-flex dark:bg-white/5"
  role="tablist"
>
  {#each tabs as t (t.key)}
    <button
      type="button"
      role="tab"
      aria-selected={tab === t.key}
      class="rounded-xl px-4 py-1.5 text-sm font-medium whitespace-nowrap transition {pill(tab === t.key)}"
      onclick={() => select('tab', t.key, 'funds')}>{t.label}</button
    >
  {/each}
</div>

{#if tab === 'funds'}
  <Card title="{fundLabel} funds" subtitle="Ranked by 12-month return; select a fund for details">
    {#snippet actions()}
      <div
        class="flex flex-wrap gap-0.5 rounded-xl bg-slate-900/5 p-0.5 dark:bg-white/5"
        role="group"
        aria-label="Fund type"
      >
        {#each FUND_TYPES as t (t.key)}
          {@const list = data.funds[t.key]}
          <button
            type="button"
            aria-pressed={fundType === t.key}
            class="rounded-lg px-2.5 py-1 text-xs font-medium transition {pill(fundType === t.key)}"
            onclick={() => select('type', t.key, 'STOCK')}
            >{t.label}{#if list.ok}<span class="ml-1 opacity-60">{list.value.length}</span>{/if}</button
          >
        {/each}
      </div>
    {/snippet}
    {#if funds.ok}
      <FundTable funds={funds.value} label={fundLabel} />
    {:else}
      <SectionError title="{fundLabel} funds are unavailable" message={funds.error} />
    {/if}
  </Card>
{:else if tab === 'bank'}
  <Card title="Bank savings rates" subtitle="Highest published deposit rate per bank, % per year">
    {#if data.bankRates.ok}
      <BankRatesChart rates={data.bankRates.value} />
    {:else}
      <SectionError title="Bank rates are unavailable" message={data.bankRates.error} />
    {/if}
  </Card>
{:else}
  <GoldPanel />
{/if}
