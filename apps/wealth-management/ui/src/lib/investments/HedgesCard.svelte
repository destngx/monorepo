<script lang="ts">
  import type { Hedges } from '$lib/api/types';
  import { GOLD, USD } from '$lib/charts/palette';
  import TrendLines from '$lib/charts/TrendLines.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { formatDate, formatPercent, formatSignedPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { HEDGE_RANGES, type HedgeRange, hedgeYearText, rebase, trendChange } from './exposure';
  import { nf } from './holdings';

  // ROADMAP M4: gold and the US dollar as ways to hold value outside VND, against the cash share.
  // Protection options, not trading charts: daily closes rebased to 0% on one percent axis.
  let { hedges }: { hedges: Hedges } = $props();

  let range = $state<HedgeRange>('1Y');
  const yearText = $derived(hedgeYearText(hedges));
  const trends = $derived(
    [
      { trend: hedges.gold, color: GOLD },
      { trend: hedges.usd, color: USD },
    ].filter((t) => t.trend !== null),
  );
  const series = $derived(
    trends.map(({ trend, color }) => ({ name: trend!.label, color, points: rebase(trend!, range) })),
  );
</script>

<Card title="Gold and USD as protection" subtitle="Ways to hold value outside VND, against your cash">
  {#snippet actions()}
    <div class="flex gap-1" role="group" aria-label="Range">
      {#each HEDGE_RANGES as option (option)}
        <button
          type="button"
          class="rounded-md px-2 py-0.5 text-xs {range === option
            ? 'bg-indigo-500/15 font-semibold text-indigo-700 dark:text-indigo-300'
            : 'muted'}"
          aria-pressed={range === option}
          onclick={() => (range = option)}>{option}</button
        >
      {/each}
    </div>
  {/snippet}

  <div class="border-l-4 pl-3 {STATUS.info.edge}">
    <p class="text-lg" data-testid="hedges-headline">
      Your cash is <strong class="tabular">{formatPercent(hedges.cashShare, 0)}</strong> of your assets, all in VND{yearText
        ? `; ${yearText}`
        : ''}.
    </p>
    <p class="muted mt-1 text-xs">
      Everything but crypto is in VND: {formatPercent(hedges.vndShare, 0)} of your assets. Gold is the SJC sell price; buying
      and selling gold or dollars costs a spread.
    </p>
  </div>

  <div class="mt-4">
    <TrendLines {series} height={200} label="Gold and USD/VND, percent change over {range}" />
  </div>

  {#if trends.length}
    <table class="mt-3 w-full text-sm">
      <thead>
        <tr class="muted text-left text-xs uppercase">
          <th class="py-2">Asset</th>
          <th class="py-2 text-right">Latest</th>
          <th class="py-2 text-right">3M</th>
          <th class="py-2 text-right">1Y</th>
          <th class="py-2 text-right">Your cash held in it (1Y)</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each trends as { trend, color } (trend!.asset)}
          {@const t = trend!}
          {@const quarter = trendChange(t, '3M')}
          {@const twelve = trendChange(t, '1Y')}
          <tr>
            <td class="py-2">
              <span class="flex items-center gap-2">
                <span class="inline-block h-0.5 w-4 rounded" style:background-color={color}></span>{t.label}
              </span>
              {#if t.source}<span class="block text-xs"><SourceBadge source={t.source} /></span>{/if}
            </td>
            <td class="py-2 text-right">
              <span class="tabular">{nf.format(t.latest)}</span>
              <span class="muted block text-xs">{t.unit}, {formatDate(t.latestDate)}</span>
            </td>
            <td class="tabular py-2 text-right">{formatSignedPercent(quarter?.changePct, 1)}</td>
            <td class="tabular py-2 text-right">
              {formatSignedPercent(twelve?.changePct, 1)}
              {#if twelve?.partial}<span class="muted block text-xs">since {formatDate(twelve.fromDate)}</span>{/if}
            </td>
            <td class="py-2 text-right">
              {#if twelve}<Money value={twelve.onCash} signed tone />{:else}<span class="muted">-</span>{/if}
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
    <p class="muted mt-2 text-xs">
      The last column is what your VND cash (<Money value={hedges.cash} />) would have gained or lost had it been held
      in the asset for the period. Past moves, not a forecast.
    </p>
  {/if}
  {#if hedges.issues.length}
    <ul class="muted mt-2 list-disc pl-5 text-xs">
      {#each hedges.issues as issue, i (i)}<li>{issue}</li>{/each}
    </ul>
  {/if}
</Card>
