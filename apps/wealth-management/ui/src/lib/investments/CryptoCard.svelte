<script lang="ts">
  import type { CryptoExposure } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { formatPercent, formatSignedPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { drawdownStatus } from './exposure';
  import { nf, quantity } from './holdings';

  // ROADMAP M2: crypto is long-term holding and trading, outside the goal pool (D32). The headline
  // is its weight in net worth and what a fixed fall would cost; no forecast.
  let { crypto, netWorth }: { crypto: CryptoExposure; netWorth: number } = $props();

  const first = $derived(crypto.scenarios[0] ?? null);
  const firstStatus = $derived(first ? drawdownStatus(first) : null);
</script>

<Card title="Crypto" subtitle="Long-term holding and trading, outside the goal pool">
  {#snippet actions()}
    {#if firstStatus}<Badge status={firstStatus.tone}>{firstStatus.label}</Badge>{/if}
  {/snippet}

  {#if crypto.value <= 0}
    <p class="muted text-sm">No crypto holdings or exchange balances.</p>
  {:else}
    <div class="border-l-4 pl-3 {firstStatus ? STATUS[firstStatus.tone].edge : STATUS.neutral.edge}">
      <p class="text-lg" data-testid="crypto-headline">
        Crypto is <strong><Money value={crypto.value} /></strong>,
        {#if crypto.shareOfNetWorth !== null}
          <strong class="tabular">{formatPercent(crypto.shareOfNetWorth, 1)}</strong> of your net worth.
        {:else}
          {formatPercent(crypto.share, 1)} of your assets.
        {/if}
        {#if first}
          If crypto falls {formatPercent(first.fallPct, 0)}, net worth falls by
          <strong><Money value={first.loss} /></strong>{#if first.netWorthChangePct !== null}
            ({formatSignedPercent(first.netWorthChangePct, 1)}){/if}.
        {/if}
      </p>
      <p class="muted mt-1 text-xs">
        Valued as the larger of exchange balances <Money value={crypto.accounts} /> and holdings
        <Money value={crypto.holdings} />. Net worth <Money value={netWorth} /> is from your accounts and money lent.
      </p>
    </div>

    <table class="mt-4 w-full text-sm" aria-label="Crypto drawdown scenarios">
      <thead>
        <tr class="muted text-left text-xs uppercase">
          <th class="py-2">If crypto falls</th>
          <th class="py-2 text-right">Net worth falls by</th>
          <th class="py-2 text-right">Net worth after</th>
          <th class="py-2 text-right">Impact</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each crypto.scenarios as scenario (scenario.fallPct)}
          {@const status = drawdownStatus(scenario)}
          <tr>
            <td class="tabular py-2">{formatPercent(scenario.fallPct, 0)}</td>
            <td class="py-2 text-right">
              <Money value={scenario.loss} />
              <span class="muted block text-xs">{formatSignedPercent(scenario.netWorthChangePct, 1)}</span>
            </td>
            <td class="py-2 text-right"><Money value={scenario.netWorthAfter} /></td>
            <td class="py-2 text-right"><Badge status={status.tone}>{status.label}</Badge></td>
          </tr>
        {/each}
      </tbody>
    </table>

    {#if crypto.coins.length}
      <div class="-mx-5 mt-4 overflow-x-auto">
        <table class="w-full min-w-[40rem] text-sm" aria-label="Held coins">
          <thead>
            <tr class="muted text-left text-xs uppercase">
              <th class="px-5 py-2">Coin</th>
              <th class="px-5 py-2 text-right">Quantity</th>
              <th class="px-5 py-2 text-right">Price</th>
              <th class="px-5 py-2 text-right">Value</th>
              <th class="px-5 py-2 text-right">24h</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each crypto.coins as coin (coin.symbol)}
              <tr>
                <td class="px-5 py-2 font-medium">
                  {coin.symbol}
                  <span class="muted block text-xs font-normal">{formatPercent(coin.share, 1)} of holdings</span>
                </td>
                <td class="tabular px-5 py-2 text-right">{quantity(coin.quantity)}</td>
                <td class="px-5 py-2 text-right">
                  {#if coin.price}
                    <span class="tabular">{nf.format(coin.price.price)} {coin.price.currency}</span>
                    {#if coin.price.source}<span class="block text-xs"><SourceBadge source={coin.price.source} /></span
                      >{/if}
                  {:else}<span class="muted">-</span>{/if}
                </td>
                <td class="px-5 py-2 text-right">
                  {#if coin.value > 0}<Money value={coin.value} />{:else}<span class="muted">-</span>{/if}
                </td>
                <td class="px-5 py-2 text-right">
                  {#if coin.move && coin.change !== null}
                    <span class="tabular">{formatSignedPercent(coin.move.changePct, 2)}</span>
                    <span class="block text-xs"><Money value={coin.change} signed tone /></span>
                  {:else}
                    <span class="muted text-xs">{coin.issue ? 'no change data' : '-'}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    <p class="muted mt-2 text-xs">Scenarios are fixed falls, not forecasts.</p>
  {/if}
</Card>
