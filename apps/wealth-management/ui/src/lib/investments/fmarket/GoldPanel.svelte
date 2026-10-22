<script lang="ts">
  import { track, Card, Deferred, SkeletonChart, Spinner } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import { defer } from '$lib/api/load';
  import { PALETTE } from '$lib/charts/palette';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { sourceOf } from '$lib/source';
  import { formatDate, formatSignedPercent, formatVND, todayISO } from '$lib/format';
  import {
    downsample,
    goldChartRows,
    historyParams,
    latest,
    parseGoldHistory,
    parseGoldSnapshot,
    parseUsdHistory,
    sourced,
    worldGoldMillionVND,
    type Range,
  } from '../fmarket';
  import MultiLineChart from './MultiLineChart.svelte';
  import RangeButtons from './RangeButtons.svelte';

  // Gold and USD share one range so world gold converts at the same day's
  // rate. Both stream in (WM-UI-16); another range keeps the shown charts with
  // a Refreshing badge until the new ones arrive.
  let range = $state<Range>('1Y');
  let reloads = $state(0);
  const requests = $derived.by(() => {
    const selected = range;
    void reloads;
    const params = historyParams(selected, todayISO());
    return {
      range: selected,
      gold: defer(async () => {
        const payload = await api.fmarket<unknown>('getGoldPriceHistory', params);
        return { rows: parseGoldHistory(payload), snapshot: parseGoldSnapshot(payload), source: sourceOf(payload) };
      }),
      usd: defer(async () => sourced(await api.fmarket<unknown>('getUsdRateHistory', params), parseUsdHistory)),
    };
  });
  const goldTrack = track(() => requests.gold);
  const usdTrack = track(() => requests.usd);
  const gold = $derived(goldTrack.value);
  const usd = $derived(usdTrack.value);
  const loading = $derived(goldTrack.pending || usdTrack.pending);
  // fmarket's change percents cover the requested range, so label them with it.
  let loadedRange = $state<Range>('1Y');
  $effect(() => {
    const { range: selected, gold: request } = requests;
    void request.then((result) => {
      if (result.ok && request === requests.gold) loadedRange = selected;
    });
  });
  const retry = () => reloads++;

  const usdPoints = $derived(usd?.data ?? []);
  const goldSource = $derived(gold?.source);
  const usdSource = $derived(usd?.source);
  const snapshot = $derived(gold?.snapshot ?? null);
  const usdRate = $derived(latest(usdPoints, 'value') ?? snapshot?.usdRate ?? null);
  const chartRows = $derived(gold ? downsample(goldChartRows(gold.rows, usdPoints, snapshot?.usdRate ?? usdRate)) : []);
  const goldRows = $derived(gold?.rows ?? []);
  const sjcBid = $derived(latest(goldRows, 'sjcBid'));
  const sjcAsk = $derived(latest(goldRows, 'sjcAsk'));
  const worldUsd = $derived(latest(goldRows, 'world'));
  const worldVnd = $derived(
    snapshot?.worldVnd ?? (worldUsd !== null && usdRate ? worldGoldMillionVND(worldUsd, usdRate) * 1e6 : null),
  );
  const lastDate = $derived(goldRows.at(-1)?.date ?? null);

  const million = (value: number) => value.toLocaleString('vi-VN', { maximumFractionDigits: 1 });
  const rate = (value: number) => value.toLocaleString('vi-VN', { maximumFractionDigits: 0 });
  const colors = { sjcAsk: PALETTE[2], sjcBid: PALETTE[3], world: PALETTE[0], usd: PALETTE[4] };
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <p class="muted flex flex-wrap items-center gap-2 text-sm">
    {#if lastDate}Prices as of {formatDate(lastDate)}{/if}
    {#if loading}<span class="inline-flex items-center gap-1" role="status"
        ><Spinner size={12} />{lastDate ? 'Updating…' : 'Loading prices…'}</span
      >{/if}
    {#if goldSource}<SourceBadge source={goldSource} />{/if}
  </p>
  <RangeButtons value={range} onchange={(next) => (range = next)} />
</div>

<!-- WM-INV-10: SJC and 9999 buy/sell, world gold converted, USD/VND -->
<div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
  <Stat
    label="SJC bar (₫/lượng)"
    change={snapshot?.sjcChangePercent ?? undefined}
    hint={snapshot?.sjcChangePercent != null ? `over ${loadedRange}` : undefined}
  >
    <span class="block text-base"
      ><span class="muted text-xs font-normal">Buy</span> <span class="tabular">{formatVND(sjcBid)}</span></span
    >
    <span class="block text-base"
      ><span class="muted text-xs font-normal">Sell</span> <span class="tabular">{formatVND(sjcAsk)}</span></span
    >
  </Stat>
  <Stat label="9999 ring (₫/lượng)">
    <span class="block text-base"
      ><span class="muted text-xs font-normal">Buy</span>
      <span class="tabular">{formatVND(snapshot?.ringBid)}</span></span
    >
    <span class="block text-base"
      ><span class="muted text-xs font-normal">Sell</span>
      <span class="tabular">{formatVND(snapshot?.ringAsk)}</span></span
    >
  </Stat>
  <Stat
    label="World gold (₫/lượng)"
    change={snapshot?.goldChangePercent ?? undefined}
    hint={[
      snapshot?.goldChangePercent != null ? `over ${loadedRange}` : '',
      worldUsd !== null ? `${worldUsd.toLocaleString('en-US')} USD/oz` : '',
    ]
      .filter(Boolean)
      .join(' · ') || undefined}
  >
    <span class="tabular">{formatVND(worldVnd)}</span>
  </Stat>
  <Stat label="USD/VND">
    <span class="tabular">{usdRate === null ? '-' : rate(usdRate)}</span>
  </Stat>
</div>

<Card
  title="Gold, Vietnam vs world"
  subtitle="Million ₫ per lượng; world = USD/oz × 1.2056 × USD rate of the day"
  class="mt-4"
>
  <Deferred data={requests.gold} label="gold prices" {retry}>
    {#snippet children(value)}
      <div>
        <MultiLineChart
          label="Gold price history"
          dates={chartRows.map((row) => row.date)}
          series={[
            { name: 'SJC sell', color: colors.sjcAsk, values: chartRows.map((row) => row.sjcAsk) },
            { name: 'SJC buy', color: colors.sjcBid, values: chartRows.map((row) => row.sjcBid) },
            { name: 'World', color: colors.world, values: chartRows.map((row) => row.world) },
          ]}
          format={million}
        />
        <p class="muted mt-2 text-xs">
          fmarket publishes no 9999 ring history; its current price is shown above.
          {#if value.snapshot?.usdRate}Days before the USD history use today's rate.{/if}
        </p>
      </div>
    {/snippet}
    {#snippet skeleton()}<SkeletonChart height={224} label="gold prices" />{/snippet}
  </Deferred>
</Card>

<Card title="USD/VND" subtitle="Selling rate" class="mt-4">
  {#snippet actions()}
    {#if usdSource}<SourceBadge source={usdSource} />{/if}
  {/snippet}
  <Deferred data={requests.usd} label="USD rates" {retry}>
    {#snippet children(value)}
      {@const points = downsample(value.data)}
      <div>
        <MultiLineChart
          label="USD/VND history"
          dates={points.map((point) => point.date)}
          series={[{ name: 'USD/VND', color: colors.usd, values: points.map((point) => point.value) }]}
          format={rate}
        />
        {#if value.data.length > 1}
          {@const change = (value.data[value.data.length - 1].value / value.data[0].value - 1) * 100}
          <p class="muted mt-2 text-xs">Change over range: {formatSignedPercent(change, 2)}</p>
        {/if}
      </div>
    {/snippet}
    {#snippet skeleton()}<SkeletonChart height={224} label="USD rates" />{/snippet}
  </Deferred>
</Card>
