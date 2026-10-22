<script lang="ts">
  import { formatDate, formatSignedPercent } from '$lib/format';
  import { extent, linePath, scale, ticks } from '@libs/ui-kit';

  // Several percent-change series on one percent axis, placed by date so series with different
  // trading days line up. A legend names each series; the hover readout gives every value.
  interface Series {
    name: string;
    color: string;
    points: { date: string; value: number }[];
  }

  let { series, height = 220, label = 'Trend' }: { series: Series[]; height?: number; label?: string } = $props();

  const width = 640;
  const pad = { top: 24, right: 12, bottom: 24, left: 52 };

  const time = (date: string) => Date.parse(`${date}T00:00:00Z`);
  const drawn = $derived(series.filter((s) => s.points.length > 1));
  const allPoints = $derived(drawn.flatMap((s) => s.points));
  const xExtent = $derived(extent(allPoints.map((p) => time(p.date))));
  const yExtent = $derived(extent([0, ...allPoints.map((p) => p.value)], { includeZero: true }));
  const yTicks = $derived(ticks(yExtent));
  const x = $derived(scale(xExtent, [pad.left, width - pad.right]));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const paths = $derived(
    drawn.map((s) => ({ ...s, d: linePath(s.points.map((p) => [x(time(p.date)), y(p.value)] as [number, number])) })),
  );
  const first = $derived(allPoints.length ? allPoints.reduce((a, b) => (a.date < b.date ? a : b)).date : '');
  const last = $derived(allPoints.length ? allPoints.reduce((a, b) => (a.date > b.date ? a : b)).date : '');

  let hover = $state<number | null>(null);
  // The point of each series nearest the hovered time.
  const readout = $derived(
    hover === null
      ? []
      : drawn.map((s) => {
          const at = hover as number;
          const point = s.points.reduce((a, b) => (Math.abs(time(b.date) - at) < Math.abs(time(a.date) - at) ? b : a));
          return { name: s.name, color: s.color, point };
        }),
  );

  function onmove(event: PointerEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * width;
    const fraction = Math.max(0, Math.min(1, (px - pad.left) / (width - pad.left - pad.right)));
    hover = xExtent[0] + fraction * (xExtent[1] - xExtent[0]);
  }
</script>

{#if drawn.length}
  {#if drawn.length > 1}
    <ul class="mb-2 flex flex-wrap gap-4 text-xs" aria-label="Legend">
      {#each drawn as s (s.name)}
        <li class="flex items-center gap-1.5">
          <span class="inline-block h-0.5 w-4 rounded" style="background-color: {s.color}"></span>{s.name}
        </li>
      {/each}
    </ul>
  {/if}
  <svg
    viewBox="0 0 {width} {height}"
    class="h-auto w-full touch-none"
    role="img"
    aria-label={label}
    onpointermove={onmove}
    onpointerleave={() => (hover = null)}
  >
    {#each yTicks as tick (tick)}
      <line
        x1={pad.left}
        x2={width - pad.right}
        y1={y(tick)}
        y2={y(tick)}
        class={tick === 0 ? 'stroke-stone-500/60' : 'stroke-stone-900/10 dark:stroke-white/10'}
      />
      <text x={pad.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="fill-stone-500 text-[11px]"
        >{formatSignedPercent(tick, 0)}</text
      >
    {/each}
    <text x={pad.left} y={height - 6} text-anchor="start" class="fill-stone-500 text-[11px]">{formatDate(first)}</text>
    <text x={width - pad.right} y={height - 6} text-anchor="end" class="fill-stone-500 text-[11px]"
      >{formatDate(last)}</text
    >
    {#each paths as s (s.name)}
      <path d={s.d} fill="none" stroke={s.color} stroke-width="2" stroke-linejoin="round" />
    {/each}
    {#if hover !== null}
      <line
        x1={x(hover)}
        x2={x(hover)}
        y1={pad.top}
        y2={height - pad.bottom}
        class="stroke-stone-400/60"
        stroke-dasharray="3 3"
      />
      {#each readout as r (r.name)}
        <circle cx={x(time(r.point.date))} cy={y(r.point.value)} r="4" fill={r.color} />
      {/each}
      <text x={pad.left} y={12} class="fill-stone-700 text-[12px] font-medium dark:fill-stone-200">
        {formatDate(readout[0]?.point.date)} · {readout
          .map((r) => `${r.name} ${formatSignedPercent(r.point.value)}`)
          .join(' · ')}
      </text>
    {/if}
  </svg>
{:else}
  <p class="muted py-8 text-center text-sm">Not enough history to draw a trend yet.</p>
{/if}
