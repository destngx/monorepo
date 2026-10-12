<script lang="ts">
  import type { HoldingSection } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import { sectionTotal, toVND, type Rates } from '$lib/investments/valuation';

  let { data } = $props();

  // USDT/VND arrives with the market phase (WM-MKT-2); until then USDT stays unconverted.
  const rates = $derived<Rates>(data.usdRate ? { USD: data.usdRate.rate } : {});
  const sections = $derived([
    { key: 'crypto', title: 'Crypto', section: data.assets.crypto },
    { key: 'funds', title: 'Fund certificates', section: data.assets.funds },
  ]);
  const totals = $derived(sections.map((s) => sectionTotal(s.section.items, rates)));
  const grand = $derived(totals.reduce((sum, t) => sum + t.vnd, 0));
  const unvalued = $derived(totals.reduce((sum, t) => sum + t.unvalued, 0));

  const nf = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 8 });
  const quantity = (value: number | null) => (value === null ? '-' : nf.format(value));
  const visible = (section: HoldingSection) => section.items.filter((h) => h.symbol || h.value !== null);

  const bankRates = $derived([...data.bankRates].sort((a, b) => b.rate - a.rate).slice(0, 15));
</script>

<PageHeader title="Investments" subtitle="Holdings from the sheet, valued in VND" />

{#if data.marketError}
  <div class="glass mb-4 border border-amber-500/40 p-3 text-sm text-amber-800 dark:text-amber-200" role="status">
    {data.marketError}
  </div>
{/if}

<!-- WM-INV-4 -->
<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Stat label="Portfolio (VND)" value={grand} hint={unvalued ? `${unvalued} holdings not valued` : undefined} />
  <Stat label="Crypto" value={totals[0].vnd} />
  <Stat label="Fund certificates" value={totals[1].vnd} />
  <Stat label="USD/VND">
    {#if data.usdRate}
      <span class="tabular">{nf.format(data.usdRate.rate)}</span>
      <span class="muted block text-xs font-normal"
        >{data.usdRate.provider}{data.usdRate.timestamp ? ` · ${formatDate(data.usdRate.timestamp)}` : ''}</span
      >
    {:else}<span class="muted">-</span>{/if}
  </Stat>
</div>

{#each sections as { key, title, section }, index (key)}
  {@const rows = visible(section)}
  <Card {title} subtitle="{rows.length} holdings" class="mt-4">
    {#if rows.length}
      <div class="-mx-5 overflow-x-auto">
        <table class="w-full min-w-[44rem] text-sm">
          <thead>
            <tr class="muted text-left text-xs uppercase">
              <th class="px-5 py-2">Platform</th>
              <th class="px-5 py-2">Symbol</th>
              <th class="px-5 py-2 text-right">Quantity</th>
              <th class="px-5 py-2 text-right">Price</th>
              <th class="px-5 py-2 text-right">Value</th>
              <th class="px-5 py-2 text-right">Est. VND</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each rows as holding, i (i)}
              {@const vnd = toVND(holding.value, holding.currency, rates)}
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
                  {#if holding.value !== null}
                    <Money value={holding.value} currency={holding.currency} />
                    {#if holding.valueSource === 'quantity_x_price'}<Badge>qty × price</Badge>{/if}
                  {:else}<span class="muted">No value</span>{/if}
                </td>
                <td class="px-5 py-2 text-right"
                  >{#if vnd !== null}<Money value={vnd} />{:else}<span class="muted">-</span>{/if}</td
                >
              </tr>
            {/each}
          </tbody>
          <tfoot>
            <tr class="border-t border-slate-900/10 font-semibold dark:border-white/10">
              <td class="px-5 py-2" colspan="5">Total</td>
              <td class="px-5 py-2 text-right"><Money value={totals[index].vnd} /></td>
            </tr>
          </tfoot>
        </table>
      </div>
    {:else}
      <EmptyState title="No {title.toLowerCase()} holdings" />
    {/if}
  </Card>
{/each}

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
