<script lang="ts">
  import type { Metric, TrendPoint } from '$lib/api/types';
  import FlagTag from '$lib/components/FlagTag.svelte';
  import { formatDate } from '$lib/format';
  import { formatNumber, pointRange, pointSource, pointValue } from '$lib/metrics';
  import { dateTicks, extent, fromDay, linePath, scale, ticks, toDay } from './geometry';
  import { FLAGGED, SERIES } from './palette';

  // One metric over time (HW-MET-3): values in the canonical unit on a time
  // axis, each paper's own range shaded behind its point, the paper's flag
  // as the marker's shape. Points without a number are left to the table.
  interface Props {
    points: TrendPoint[];
    metric: Metric;
    /** Opens the paper a point came from. */
    onopen?: (point: TrendPoint) => void;
  }

  let { points, metric, onopen }: Props = $props();

  // Drawn at the box's own pixel width, so text stays 11px on a phone and on a wide screen.
  let boxWidth = $state(0);
  const width = $derived(Math.max(boxWidth || 640, 280));
  const height = 240;
  const pad = { top: 16, right: 16, bottom: 28, left: 44 };
  const plot = $derived({ left: pad.left, right: width - pad.right, top: pad.top, bottom: height - pad.bottom });

  const numeric = $derived(points.filter((p) => p.value !== undefined));
  const days = $derived(numeric.map((p) => toDay(p.date)));
  const xDomain = $derived.by((): [number, number] => {
    const first = days[0] ?? 0;
    const last = days.at(-1) ?? first;
    // One day alone sits in the middle of a month.
    return last - first < 1 ? [first - 15, last + 15] : [first, last];
  });
  const x = $derived(scale(xDomain, [plot.left + 10, plot.right - 10]));
  const yExtent = $derived(
    extent(
      numeric.flatMap((p) => [p.value as number, p.refLow, p.refHigh].filter((v): v is number => v !== undefined)),
      0.08,
    ),
  );
  const y = $derived(scale(yExtent, [plot.bottom, plot.top]));
  const yTicks = $derived(ticks(yExtent));
  const xTicks = $derived(dateTicks(xDomain[0], xDomain[1], numeric.length > 1 ? (width < 480 ? 2 : 4) : 1));
  const coords = $derived(numeric.map((p, i) => [x(days[i]), y(p.value as number)] as [number, number]));

  // Each band reaches halfway to its neighbours, so a change of lab shows as
  // a step in the shaded range; a 2px gap keeps neighbouring bands apart.
  const bands = $derived(
    numeric.flatMap((p, i) => {
      if (p.refLow === undefined && p.refHigh === undefined) return [];
      const cx = coords[i][0];
      let left = i === 0 ? plot.left : (coords[i - 1][0] + cx) / 2 + 1;
      let right = i === numeric.length - 1 ? plot.right : (cx + coords[i + 1][0]) / 2 - 1;
      if (right - left < 8) [left, right] = [cx - 4, cx + 4];
      const top = p.refHigh !== undefined ? Math.max(y(p.refHigh), plot.top) : plot.top;
      const bottom = p.refLow !== undefined ? Math.min(y(p.refLow), plot.bottom) : plot.bottom;
      return bottom > top ? [{ key: i, x: left, y: top, w: right - left, h: bottom - top }] : [];
    }),
  );

  /** A flagged point's marker: up for high, down for low, a diamond for abnormal. */
  function marker(flag: string, cx: number, cy: number): string {
    const s = 6;
    if (flag === 'H') return `M${cx},${cy - s} L${cx + s},${cy + s * 0.8} L${cx - s},${cy + s * 0.8} Z`;
    if (flag === 'L') return `M${cx},${cy + s} L${cx + s},${cy - s * 0.8} L${cx - s},${cy - s * 0.8} Z`;
    return `M${cx},${cy - s} L${cx + s},${cy} L${cx},${cy + s} L${cx - s},${cy} Z`;
  }

  let hover = $state<number | null>(null);
  const hovered = $derived(hover !== null ? numeric[hover] : undefined);

  // The wrapper has the svg's exact size, so its box maps to the viewBox.
  function onmove(event: PointerEvent) {
    const rect = (event.currentTarget as HTMLElement).getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * width;
    let best = 0;
    coords.forEach(([cx], i) => {
      if (Math.abs(cx - px) < Math.abs(coords[best][0] - px)) best = i;
    });
    hover = coords.length ? best : null;
  }

  function onkeydown(event: KeyboardEvent) {
    if (!numeric.length) return;
    if (event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
      event.preventDefault();
      const step = event.key === 'ArrowRight' ? 1 : -1;
      hover =
        hover === null ? (step > 0 ? 0 : numeric.length - 1) : Math.max(0, Math.min(numeric.length - 1, hover + step));
    } else if (event.key === 'Enter' && hovered?.documentId) {
      onopen?.(hovered);
    } else if (event.key === 'Escape') {
      hover = null;
    }
  }

  const summary = $derived.by(() => {
    if (!numeric.length) return `${metric.name}: no values to chart`;
    const last = numeric.at(-1) as TrendPoint;
    return `${metric.name}, ${numeric.length} values from ${formatDate(numeric[0].date)} to ${formatDate(last.date)}; latest ${formatNumber(last.value as number)} ${metric.unit}`;
  });
</script>

{#if numeric.length}
  <!-- The group is focusable and read with the arrow keys; the table below holds every value too. -->
  <!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
  <div
    role="group"
    aria-label="{summary}. Use the arrow keys to read each value and Enter to open its paper."
    tabindex="0"
    bind:clientWidth={boxWidth}
    class="relative touch-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ink-900 dark:focus-visible:outline-ink-100"
    class:cursor-pointer={hovered?.documentId && onopen}
    onpointermove={onmove}
    onpointerleave={(event) => event.pointerType !== 'touch' && (hover = null)}
    onclick={() => hovered?.documentId && onopen?.(hovered)}
    {onkeydown}
  >
    <svg viewBox="0 0 {width} {height}" {width} {height} class="block" role="img" aria-label={summary}>
      {#each bands as b (b.key)}
        <rect x={b.x} y={b.y} width={b.w} height={b.h} class="fill-ink-500/12 dark:fill-ink-300/12" />
      {/each}
      {#each yTicks as tick (tick)}
        <line
          x1={plot.left}
          x2={plot.right}
          y1={y(tick)}
          y2={y(tick)}
          class="stroke-ink-900/10 dark:stroke-ink-50/10"
        />
        <text x={plot.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="tabular fill-ink-500 text-[11px]"
          >{formatNumber(tick)}</text
        >
      {/each}
      {#each xTicks as tick, i (tick)}
        <text
          x={x(tick)}
          y={height - 8}
          text-anchor={xTicks.length === 1 ? 'middle' : i === 0 ? 'start' : i === xTicks.length - 1 ? 'end' : 'middle'}
          class="tabular fill-ink-500 text-[11px]">{formatDate(fromDay(tick))}</text
        >
      {/each}
      {#if hover !== null}
        <line x1={coords[hover][0]} x2={coords[hover][0]} y1={plot.top} y2={plot.bottom} class="stroke-ink-500/60" />
      {/if}
      {#if coords.length > 1}
        <path
          d={linePath(coords)}
          fill="none"
          stroke={SERIES}
          stroke-width="2"
          stroke-linejoin="round"
          stroke-linecap="round"
        />
      {/if}
      {#each numeric as p, i (i)}
        {@const [cx, cy] = coords[i]}
        {#if p.flag}
          <path
            d={marker(p.flag, cx, cy)}
            fill={FLAGGED}
            stroke="var(--surface)"
            stroke-width="2"
            paint-order="stroke"
          />
        {:else if p.source === 'computed'}
          <circle {cx} {cy} r="4" fill="var(--surface)" stroke={SERIES} stroke-width="2" />
        {:else}
          <circle
            {cx}
            {cy}
            r={hover === i ? 5 : 4}
            fill={SERIES}
            stroke="var(--surface)"
            stroke-width="2"
            paint-order="stroke"
          />
        {/if}
      {/each}
    </svg>
    {#if hover !== null && hovered}
      {@const v = pointValue(hovered, metric)}
      <div
        class="panel-strong pointer-events-none absolute z-10 w-max max-w-60 -translate-x-1/2 px-3 py-2 text-xs"
        style:left="{(Math.min(Math.max(coords[hover][0], 124), width - 124) / width) * 100}%"
        style:bottom={coords[hover][1] > 100 ? `${height - coords[hover][1] + 14}px` : undefined}
        style:top={coords[hover][1] > 100 ? undefined : `${coords[hover][1] + 14}px`}
        data-testid="trend-readout"
      >
        <p class="flex items-center gap-2">
          <span class="text-sm font-semibold tabular">{v.value}</span><span class="muted">{v.unit}</span>
          <FlagTag flag={hovered.flag} />
        </p>
        <p class="muted mt-0.5">{formatDate(hovered.date)} · {pointSource(hovered)}</p>
        {#if pointRange(hovered)}<p class="muted">Paper's range {pointRange(hovered)}</p>{/if}
        {#if hovered.printedUnit && hovered.printedUnit !== metric.unit}
          <p class="muted">Printed as {hovered.printed} {hovered.printedUnit}</p>
        {/if}
      </div>
    {/if}
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">No numeric values to chart. Each result is listed below.</p>
{/if}
