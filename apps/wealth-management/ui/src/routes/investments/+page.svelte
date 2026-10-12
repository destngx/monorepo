<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { PortfolioSection, PortfolioValuation, ValuedHolding } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
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

  const nf = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 8 });
  const quantity = (value: number | null) => (value === null ? '-' : nf.format(value));
  const source = (h: ValuedHolding) =>
    ({ quantity_x_price: 'qty × sheet price', quantity_x_live_price: 'qty × live price' })[h.valueSource];

  const bankRates = $derived([...data.bankRates].sort((a, b) => b.rate - a.rate).slice(0, 15));
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
        <span class="muted block text-xs font-normal"
          >{rate.provider}{rate.timestamp ? ` · ${formatDate(rate.timestamp)}` : ''}</span
        >
      {/each}
    {:else}<span class="muted">-</span>{/if}
  </Stat>
</div>

{#each sections as { key, title, rows, total } (key)}
  <Card {title} subtitle="{rows.length} holdings" class="mt-4">
    {#if rows.length}
      <div class="-mx-5 overflow-x-auto">
        <table class="w-full min-w-[52rem] text-sm">
          <thead>
            <tr class="muted text-left text-xs uppercase">
              <th class="px-5 py-2">Platform</th>
              <th class="px-5 py-2">Symbol</th>
              <th class="px-5 py-2 text-right">Quantity</th>
              <th class="px-5 py-2 text-right">Sheet price</th>
              <th class="px-5 py-2 text-right">Live price</th>
              <th class="px-5 py-2 text-right">Value</th>
              <th class="px-5 py-2 text-right">VND</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each rows as holding, i (i)}
              {@const label = source(holding)}
              <tr>
                <td class="px-5 py-2">{holding.platform}</td>
                <td class="px-5 py-2 font-medium">{holding.symbol || '-'}</td>
                <td class="tabular px-5 py-2 text-right">{quantity(holding.quantity)}</td>
                <td class="px-5 py-2 text-right"
                  >{#if holding.price !== null}<Money value={holding.price} currency={holding.currency} />{:else}<span
                      class="muted">-</span
                    >{/if}</td
                >
                <td class="px-5 py-2 text-right">
                  {#if holding.livePrice}
                    <Money value={holding.livePrice.price} currency={holding.livePrice.currency} />
                    <span class="muted block text-xs"
                      >{holding.livePrice.provider}{holding.livePrice.timestamp
                        ? ` · ${formatDate(holding.livePrice.timestamp)}`
                        : ''}</span
                    >
                  {:else}<span class="muted">-</span>{/if}
                </td>
                <td class="px-5 py-2 text-right">
                  {#if holding.value !== null}
                    <Money value={holding.value} currency={holding.currency} />
                    {#if label}<Badge tone={holding.valueSource === 'quantity_x_live_price' ? 'blue' : 'neutral'}
                        >{label}</Badge
                      >{/if}
                  {:else}<span class="muted">No value</span>{/if}
                </td>
                <td class="px-5 py-2 text-right">
                  {#if holding.valueVND !== null}<Money value={holding.valueVND} />{:else}
                    <span class="text-xs text-amber-700 dark:text-amber-300">{holding.issue || 'Not valued'}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
          <tfoot>
            <tr class="border-t border-slate-900/10 font-semibold dark:border-white/10">
              <td class="px-5 py-2" colspan="6">Total</td>
              <td class="px-5 py-2 text-right"><Money value={total} /></td>
            </tr>
          </tfoot>
        </table>
      </div>
    {:else}
      <EmptyState title="No {title.toLowerCase()} holdings" />
    {/if}
  </Card>
{/each}

<p class="muted mt-2 text-right text-xs">Valued {formatDate(valuation.generatedAt)}</p>

<Card title="Bank savings rates" subtitle="Highest published rates" class="mt-4">
  {#if bankRates.length}
    <ul class="grid gap-x-6 gap-y-1 text-sm sm:grid-cols-2 lg:grid-cols-3">
      {#each bankRates as rate, i (i)}
        <li class="flex justify-between gap-3 border-b border-slate-900/5 py-1.5 dark:border-white/5">
          <span>{rate.bank}{rate.tenor ? ` · ${rate.tenor}` : ''}</span>
          <span class="tabular font-medium">{formatPercent(rate.rate, 2)}</span>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted text-sm">Bank rates are unavailable right now.</p>
  {/if}
</Card>
