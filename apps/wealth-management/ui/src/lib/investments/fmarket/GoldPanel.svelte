<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import { PALETTE } from '$lib/charts/palette';
  import Card from '$lib/components/Card.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatSignedPercent, formatVND, todayISO } from '$lib/format';
  import {
    downsample,
    goldChartRows,
    historyParams,
    latest,
    parseGoldHistory,
    parseGoldSnapshot,
    parseUsdHistory,
    settle,
    worldGoldMillionVND,
    type GoldRow,
    type GoldSnapshot,
    type Point,
    type Range,
    type Section,
  } from '../fmarket';
  import MultiLineChart from './MultiLineChart.svelte';
  import RangeButtons from './RangeButtons.svelte';
  import SectionError from './SectionError.svelte';

  // Gold and USD share one range so world gold converts at the same day's rate.
  let range = $state<Range>('1Y');
  let gold = $state<Section<{ rows: GoldRow[]; snapshot: GoldSnapshot }> | null>(null);
  let usd = $state<Section<Point[]> | null>(null);
  let loading = $state(false);
  // fmarket's change percents cover the requested range, so label them with it.
  let loadedRange = $state<Range>('1Y');

  async function load(selected: Range, signal: AbortSignal) {
    loading = true;
    const params = historyParams(selected, todayISO());
    const [goldResult, usdResult] = await Promise.all([
      settle(async () => {
        const payload = await api.fmarket<unknown>('getGoldPriceHistory', params, { signal });
        return { rows: parseGoldHistory(payload), snapshot: parseGoldSnapshot(payload) };
      }),
      settle(async () => parseUsdHistory(await api.fmarket<unknown>('getUsdRateHistory', params, { signal }))),
    ]);
    // A newer range (or leaving the page) aborted this request.
    if (signal.aborted) return;
    gold = goldResult;
    loadedRange = selected;
    usd = usdResult;
    loading = false;
  }

  $effect(() => {
    const controller = new AbortController();
    void load(range, controller.signal);
    return () => controller.abort();
  });

  const usdPoints = $derived(usd?.ok ? usd.value : []);
  const snapshot = $derived(gold?.ok ? gold.value.snapshot : null);
  const usdRate = $derived(latest(usdPoints, 'value') ?? snapshot?.usdRate ?? null);
  const chartRows = $derived(
    gold?.ok ? downsample(goldChartRows(gold.value.rows, usdPoints, snapshot?.usdRate ?? usdRate)) : [],
  );
  const goldRows = $derived(gold?.ok ? gold.value.rows : []);
  const sjcBid = $derived(latest(goldRows, 'sjcBid'));
  const sjcAsk = $derived(latest(goldRows, 'sjcAsk'));
  const worldUsd = $derived(latest(goldRows, 'world'));
  const worldVnd = $derived(
    snapshot?.worldVnd ?? (worldUsd !== null && usdRate ? worldGoldMillionVND(worldUsd, usdRate) * 1e6 : null),
  );
  const lastDate = $derived(goldRows.at(-1)?.date ?? null);
  const usdChartPoints = $derived(downsample(usdPoints));

  const million = (value: number) => value.toLocaleString('vi-VN', { maximumFractionDigits: 1 });
  const rate = (value: number) => value.toLocaleString('vi-VN', { maximumFractionDigits: 0 });
  const colors = { sjcAsk: PALETTE[2], sjcBid: PALETTE[3], world: PALETTE[0], usd: PALETTE[4] };
</script>

<div class="mb-4 flex flex-wrap items-center justify-between gap-3">
  <p class="muted text-sm">
    {#if lastDate}Prices as of {formatDate(lastDate)}{:else if loading}Loading prices…{/if}
  </p>
  <RangeButtons value={range} onchange={(next) => (range = next)} disabled={loading} />
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
  {#if gold && !gold.ok}
    <SectionError title="Gold prices are unavailable" message={gold.error} />
  {:else if !gold}
    <div class="h-56 animate-pulse rounded-xl bg-slate-900/5 dark:bg-white/5"></div>
  {:else}
    <div class:opacity-50={loading} class="transition-opacity">
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
    </div>
    <p class="muted mt-2 text-xs">
      fmarket publishes no 9999 ring history; its current price is shown above.
      {#if snapshot?.usdRate}Days before the USD history use today's rate.{/if}
    </p>
  {/if}
</Card>

<Card title="USD/VND" subtitle="Selling rate" class="mt-4">
  {#if usd && !usd.ok}
    <SectionError title="USD rates are unavailable" message={usd.error} />
  {:else if !usd}
    <div class="h-56 animate-pulse rounded-xl bg-slate-900/5 dark:bg-white/5"></div>
  {:else}
    <div class:opacity-50={loading} class="transition-opacity">
      <MultiLineChart
        label="USD/VND history"
        dates={usdChartPoints.map((point) => point.date)}
        series={[{ name: 'USD/VND', color: colors.usd, values: usdChartPoints.map((point) => point.value) }]}
        format={rate}
      />
    </div>
    {#if usdPoints.length > 1}
      {@const change = (usdPoints[usdPoints.length - 1].value / usdPoints[0].value - 1) * 100}
      <p class="muted mt-2 text-xs">Change over range: {formatSignedPercent(change, 2)}</p>
    {/if}
  {/if}
</Card>
