<script lang="ts">
  import { formatCompact, formatDate, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { extent, linePath, scale, ticks } from './geometry';

  interface Props {
    points: { date: string; value: number }[];
    height?: number;
    label?: string;
  }

  let { points, height = 220, label = 'Trend' }: Props = $props();

  const width = 640;
  const pad = { top: 12, right: 12, bottom: 24, left: 56 };

  const values = $derived(points.map((p) => p.value));
  const yExtent = $derived(extent(values));
  const yTicks = $derived(ticks(yExtent));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const x = $derived(scale([0, Math.max(points.length - 1, 1)], [pad.left, width - pad.right]));
  const coords = $derived(points.map((p, i) => [x(i), y(p.value)] as [number, number]));
  const area = $derived(
    coords.length
      ? `${linePath(coords)} L${coords[coords.length - 1][0]},${height - pad.bottom} L${coords[0][0]},${height - pad.bottom} Z`
      : '',
  );
  const xLabels = $derived(
    points.length > 1 ? [0, Math.floor((points.length - 1) / 2), points.length - 1] : points.length ? [0] : [],
  );

  let hover = $state<number | null>(null);

  function onmove(event: PointerEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * width;
    const index = Math.round(((px - pad.left) / (width - pad.left - pad.right)) * (points.length - 1));
    hover = Math.max(0, Math.min(points.length - 1, index));
  }
</script>

{#if points.length > 1}
  <svg
    viewBox="0 0 {width} {height}"
    class="h-auto w-full touch-none"
    role="img"
    aria-label={label}
    onpointermove={onmove}
    onpointerleave={() => (hover = null)}
  >
    <defs>
      <linearGradient id="line-fill" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0%" stop-color="#6366f1" stop-opacity="0.3" />
        <stop offset="100%" stop-color="#6366f1" stop-opacity="0" />
      </linearGradient>
    </defs>
    {#each yTicks as tick (tick)}
      <line
        x1={pad.left}
        x2={width - pad.right}
        y1={y(tick)}
        y2={y(tick)}
        class="stroke-slate-900/10 dark:stroke-white/10"
      />
      <text x={pad.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="fill-slate-500 text-[11px]"
        >{mask.revealed ? formatCompact(tick) : '•••'}</text
      >
    {/each}
    {#each xLabels as index (index)}
      <text
        x={x(index)}
        y={height - 6}
        text-anchor={index === 0 ? 'start' : index === points.length - 1 ? 'end' : 'middle'}
        class="fill-slate-500 text-[11px]">{formatDate(points[index].date)}</text
      >
    {/each}
    <path d={area} fill="url(#line-fill)" />
    <path d={linePath(coords)} fill="none" stroke="#6366f1" stroke-width="2" stroke-linejoin="round" />
    {#if hover !== null}
      <line
        x1={coords[hover][0]}
        x2={coords[hover][0]}
        y1={pad.top}
        y2={height - pad.bottom}
        class="stroke-slate-400/60"
        stroke-dasharray="3 3"
      />
      <circle cx={coords[hover][0]} cy={coords[hover][1]} r="4" fill="#6366f1" />
      <text
        x={Math.min(Math.max(coords[hover][0], pad.left + 60), width - 70)}
        y={pad.top + 4}
        text-anchor="middle"
        class="fill-slate-700 text-[12px] font-medium dark:fill-slate-200"
      >
        {formatDate(points[hover].date)} · {mask.revealed ? formatCompact(points[hover].value) : MASK}
      </text>
    {/if}
  </svg>
{:else}
  <p class="muted py-8 text-center text-sm">Not enough history to draw a trend yet.</p>
{/if}
