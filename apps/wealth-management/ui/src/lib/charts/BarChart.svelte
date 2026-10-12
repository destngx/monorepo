<script lang="ts">
  import { formatCompact, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { extent, scale, ticks } from './geometry';

  interface Series {
    name: string;
    color: string;
    values: number[];
  }

  interface Props {
    labels: string[];
    series: Series[];
    height?: number;
    label?: string;
  }

  // Grouped bars, e.g. income vs expense per month (WM-DASH-3).
  let { labels, series, height = 220, label = 'Bar chart' }: Props = $props();

  const width = 640;
  const pad = { top: 12, right: 8, bottom: 24, left: 56 };

  const all = $derived(series.flatMap((s) => s.values));
  const yExtent = $derived(extent(all, true));
  const yTicks = $derived(ticks(yExtent));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const band = $derived((width - pad.left - pad.right) / Math.max(labels.length, 1));
  const barWidth = $derived(Math.min(28, (band * 0.7) / Math.max(series.length, 1)));

  let hover = $state<number | null>(null);
</script>

{#if labels.length}
  <svg viewBox="0 0 {width} {height}" class="h-auto w-full" role="img" aria-label={label}>
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
    {#each labels as text, i (i)}
      {@const center = pad.left + band * i + band / 2}
      <g role="presentation" onpointerenter={() => (hover = i)} onpointerleave={() => (hover = null)}>
        <rect
          x={pad.left + band * i}
          y={pad.top}
          width={band}
          height={height - pad.top - pad.bottom}
          fill={hover === i ? 'rgb(100 116 139 / 0.08)' : 'transparent'}
        />
        {#each series as s, j (s.name)}
          {@const value = s.values[i] ?? 0}
          <rect
            x={center - (barWidth * series.length) / 2 + j * barWidth + 1}
            y={Math.min(y(value), y(0))}
            width={barWidth - 2}
            height={Math.max(Math.abs(y(value) - y(0)), 1)}
            rx="3"
            fill={s.color}
          >
            <title>{s.name}: {mask.revealed ? formatCompact(value) : MASK}</title>
          </rect>
        {/each}
        <text x={center} y={height - 6} text-anchor="middle" class="fill-slate-500 text-[11px]">{text}</text>
      </g>
    {/each}
  </svg>
  <div class="mt-2 flex flex-wrap gap-4 text-xs">
    {#each series as s (s.name)}
      <span class="muted inline-flex items-center gap-1.5"
        ><span class="size-2.5 rounded-sm" style:background={s.color}></span>{s.name}</span
      >
    {/each}
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">No data yet.</p>
{/if}
