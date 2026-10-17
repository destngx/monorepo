<script lang="ts">
  import { extent, linePath, scale, ticks } from '$lib/charts/geometry';
  import MaskedAxis from '$lib/charts/MaskedAxis.svelte';
  import Masked from '$lib/components/Masked.svelte';
  import { formatDate } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import type { ValueFormat } from '../fmarket';

  interface Series {
    name: string;
    color: string;
    /** One value per date; null leaves a gap in the line. */
    values: (number | null)[];
  }

  interface Props {
    dates: string[];
    series: Series[];
    height?: number;
    label?: string;
    format?: ValueFormat;
    /** Private amounts (e.g. card cashback): values and ticks hide with the privacy mask. */
    masked?: boolean;
  }

  // Public market prices by default, so nothing is masked unless `masked` is set.
  let {
    dates,
    series,
    height = 240,
    label = 'Price history',
    masked = false,
    format = (value: number) => value.toLocaleString('vi-VN', { maximumFractionDigits: 2 }),
  }: Props = $props();

  const width = 640;
  const pad = { top: 12, right: 12, bottom: 24, left: 60 };

  const values = $derived(series.flatMap((s) => s.values.filter((v): v is number => v !== null)));
  const yExtent = $derived(extent(values));
  const yTicks = $derived(ticks(yExtent));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const x = $derived(scale([0, Math.max(dates.length - 1, 1)], [pad.left, width - pad.right]));

  /** Line segments per series, split wherever a value is missing. */
  const paths = $derived(
    series.map((s) => {
      const segments: [number, number][][] = [];
      let current: [number, number][] = [];
      s.values.forEach((value, i) => {
        if (value === null) {
          if (current.length) segments.push(current);
          current = [];
        } else current.push([x(i), y(value)]);
      });
      if (current.length) segments.push(current);
      return { ...s, d: segments.map(linePath).join(' ') };
    }),
  );
  // First, middle and last date; with two dates the middle is the first, so it is labelled once.
  const xLabels = $derived(
    dates.length > 1
      ? [...new Set([0, Math.floor((dates.length - 1) / 2), dates.length - 1])]
      : dates.length
        ? [0]
        : [],
  );

  let hover = $state<number | null>(null);
  const shown = $derived(hover ?? dates.length - 1);

  const hidden = $derived(masked && !mask.revealed);

  function onmove(event: PointerEvent) {
    const svg = event.currentTarget as SVGSVGElement;
    const rect = svg.getBoundingClientRect();
    const px = ((event.clientX - rect.left) / rect.width) * width;
    const index = Math.round(((px - pad.left) / (width - pad.left - pad.right)) * (dates.length - 1));
    hover = Math.max(0, Math.min(dates.length - 1, index));
  }
</script>

{#if dates.length > 1 && values.length}
  <div class="mb-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-xs">
    <span class="muted tabular">{formatDate(dates[shown])}</span>
    {#each series as s (s.name)}
      {@const value = s.values[shown]}
      {@const text = value === null || value === undefined ? null : format(value)}
      <span class="inline-flex items-center gap-1.5">
        <span class="size-2.5 rounded-sm" style:background={s.color}></span>
        <span class="muted">{s.name}</span>
        <span class="tabular font-medium"
          >{#if text === null}-{:else if masked}<Masked {text} />{:else}{text}{/if}</span
        >
      </span>
    {/each}
  </div>
  <div class="relative">
    <svg
      viewBox="0 0 {width} {height}"
      class="block h-auto w-full touch-none"
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
          class="stroke-slate-900/10 dark:stroke-white/10"
        />
        <text x={pad.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="fill-slate-500 text-[11px]"
          >{hidden ? '•••' : format(tick)}</text
        >
      {/each}
      {#each xLabels as index (index)}
        <text
          x={x(index)}
          y={height - 6}
          text-anchor={index === 0 ? 'start' : index === dates.length - 1 ? 'end' : 'middle'}
          class="fill-slate-500 text-[11px]">{formatDate(dates[index])}</text
        >
      {/each}
      {#each paths as path (path.name)}
        <path d={path.d} fill="none" stroke={path.color} stroke-width="2" stroke-linejoin="round" />
      {/each}
      {#if hover !== null}
        <line
          x1={x(hover)}
          x2={x(hover)}
          y1={pad.top}
          y2={height - pad.bottom}
          class="stroke-slate-400/60"
          stroke-dasharray="3 3"
        />
        {#each series as s (s.name)}
          {@const value = s.values[hover]}
          {#if value !== null && value !== undefined}
            <circle cx={x(hover)} cy={y(value)} r="3.5" fill={s.color} />
          {/if}
        {/each}
      {/if}
    </svg>
    {#if masked}<MaskedAxis {width} {height} {pad} />{/if}
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">Not enough history for this range.</p>
{/if}
