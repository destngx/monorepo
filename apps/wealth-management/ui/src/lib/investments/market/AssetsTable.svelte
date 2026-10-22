<script lang="ts">
  import type { PulseQuote, Timeframe } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon, { type IconName } from '$lib/components/Icon.svelte';
  import { formatSignedPercent } from '$lib/format';
  import {
    TIMEFRAME_LABELS,
    changeClass,
    formatPrice,
    momentumLabel,
    sortQuotes,
    type QuoteSortKey,
    type SortDirection,
  } from '../market';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { symbolHref } from './link';

  interface Props {
    quotes: PulseQuote[];
    timeframe: Timeframe;
  }

  let { quotes, timeframe }: Props = $props();

  // Asset table sorting; null keeps the engine order.
  let sortKey = $state<QuoteSortKey | null>(null);
  let sortDirection = $state<SortDirection>('desc');
  const assets = $derived(sortQuotes(quotes, sortKey, sortDirection));
  const failed = $derived(quotes.filter((quote) => quote.error).length);

  function sortBy(key: QuoteSortKey) {
    if (sortKey === key) {
      if (sortDirection === (key === 'symbol' ? 'asc' : 'desc')) sortDirection = key === 'symbol' ? 'desc' : 'asc';
      else sortKey = null;
    } else {
      sortKey = key;
      sortDirection = key === 'symbol' ? 'asc' : 'desc';
    }
  }

  const ariaSort = (key: QuoteSortKey) =>
    sortKey === key ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none';

  const momentumIcon: Record<PulseQuote['momentum'], IconName> = { fire: 'zap', stable: 'activity', sleep: 'moon' };
  const momentumClass: Record<PulseQuote['momentum'], string> = {
    fire: 'text-orange-500',
    stable: 'text-sky-600 dark:text-sky-400',
    sleep: 'text-stone-400',
  };
  const directionRotation = { up: '', down: 'rotate-180', flat: 'rotate-90' } as const;

  const href = (symbol: string) => symbolHref(symbol, timeframe);
</script>

<!-- Sortable asset table header -->
{#snippet sortHeader(key: QuoteSortKey, label: string, align: string)}
  <th class="px-5 py-2 {align}" aria-sort={ariaSort(key)}>
    <button type="button" class="inline-flex items-center gap-1 uppercase" onclick={() => sortBy(key)}>
      {label}
      {#if sortKey === key}<Icon name="arrowUp" size={12} class={sortDirection === 'desc' ? 'rotate-180' : ''} />{/if}
    </button>
  </th>
{/snippet}

<Card
  title="Assets"
  subtitle="{TIMEFRAME_LABELS[timeframe]} change is the last bar versus the one before{failed
    ? ` · ${failed} unavailable`
    : ''}"
>
  {#if assets.length}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[52rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            {@render sortHeader('symbol', 'Asset', '')}
            <th class="px-5 py-2 text-right">Price</th>
            {@render sortHeader('changePct', TIMEFRAME_LABELS[timeframe], 'text-right')}
            {@render sortHeader('dayChangePct', 'Day', 'text-right')}
            {@render sortHeader('weekChangePct', 'Week', 'text-right')}
            <th class="px-5 py-2 text-center">Trend</th>
            <th class="px-5 py-2">Source</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
          {#each assets as quote (quote.symbol)}
            <tr class={quote.error ? 'opacity-60' : ''}>
              <td class="px-5 py-2">
                <a href={href(quote.symbol)} class="link font-medium">{quote.symbol}</a>
                <span class="muted block text-xs">{quote.name}</span>
              </td>
              {#if quote.error}
                <td class="px-5 py-2 text-xs text-rose-700 dark:text-rose-300" colspan="5">
                  <span class="inline-flex items-start gap-1"
                    ><Icon name="alert" size={14} class="mt-0.5" /> Unavailable: {quote.error}</span
                  >
                </td>
              {:else}
                <td class="tabular px-5 py-2 text-right font-medium">
                  {formatPrice(quote.price)}
                  {#if quote.currency}<span class="muted text-xs font-normal">{quote.currency}</span>{/if}
                </td>
                <td class="tabular px-5 py-2 text-right font-medium {changeClass(quote.changePct)}"
                  >{formatSignedPercent(quote.changePct, 2)}</td
                >
                <td class="tabular px-5 py-2 text-right {changeClass(quote.dayChangePct)}"
                  >{formatSignedPercent(quote.dayChangePct, 2)}</td
                >
                <td class="tabular px-5 py-2 text-right {changeClass(quote.weekChangePct)}"
                  >{formatSignedPercent(quote.weekChangePct, 2)}</td
                >
                <td class="px-5 py-2">
                  <span class="flex items-center justify-center gap-2">
                    <span class={changeClass(quote.changePct)} title="Direction: {quote.direction}">
                      <Icon name="arrowUp" size={16} class={directionRotation[quote.direction]} />
                      <span class="sr-only">{quote.direction}</span>
                    </span>
                    <span class={momentumClass[quote.momentum]} title="Momentum: {momentumLabel[quote.momentum]}">
                      <Icon name={momentumIcon[quote.momentum]} size={16} />
                      <span class="sr-only">{momentumLabel[quote.momentum]}</span>
                    </span>
                  </span>
                </td>
              {/if}
              <td class="px-5 py-2"><SourceBadge source={quote.source} /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs">
      <span class="inline-flex items-center gap-1"
        ><Icon name="zap" size={12} class="text-orange-500" /> Strong move</span
      >
      <span class="inline-flex items-center gap-1"
        ><Icon name="activity" size={12} class="text-sky-600 dark:text-sky-400" /> Steady</span
      >
      <span class="inline-flex items-center gap-1"><Icon name="moon" size={12} class="text-stone-400" /> Quiet</span>
      <span class="inline-flex items-center gap-1"
        ><Icon name="alert" size={12} class="text-amber-600" /> Served by a fallback source (hover for details)</span
      >
    </p>
  {:else}
    <EmptyState title="No assets returned" detail="No market data came back for this timeframe." />
  {/if}
</Card>
