<script lang="ts">
  import Masked from '$lib/components/Masked.svelte';
  import { formatCompact, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { extent, scale, ticks } from './geometry';
  import { leavesReadout } from './hover';
  import MaskedAxis from './MaskedAxis.svelte';

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
    /** Stack the series in one bar per label (parts of a whole) instead of grouping them. */
    stacked?: boolean;
  }

  // Grouped bars, e.g. income vs expense per month (WM-DASH-3); stacked bars for parts of a total.
  let { labels, series, height = 220, label = 'Bar chart', stacked = false }: Props = $props();

  const width = 640;
  const pad = { top: 12, right: 8, bottom: 24, left: 56 };

  // Stacked bars scale to each column's total; parts are never negative.
  const totals = $derived(labels.map((_, i) => series.reduce((sum, s) => sum + Math.max(0, s.values[i] ?? 0), 0)));
  const all = $derived(stacked ? totals : series.flatMap((s) => s.values));
  const yExtent = $derived(extent(all, true));
  const yTicks = $derived(ticks(yExtent));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const band = $derived((width - pad.left - pad.right) / Math.max(labels.length, 1));
  const barWidth = $derived(Math.min(28, (band * 0.7) / Math.max(stacked ? 1 : series.length, 1)));

  /** Bottom of series j in column i: the parts of the series below it. */
  function stackBase(i: number, j: number): number {
    let base = 0;
    for (let k = 0; k < j; k++) base += Math.max(0, series[k].values[i] ?? 0);
    return base;
  }

  let hover = $state<number | null>(null);
</script>

{#if labels.length}
  <div role="presentation" onpointerleave={(event) => leavesReadout(event) && (hover = null)}>
    <div class="relative">
      <svg viewBox="0 0 {width} {height}" class="block h-auto w-full" role="img" aria-label={label}>
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
          <g role="presentation" onpointerenter={() => (hover = i)}>
            <rect
              x={pad.left + band * i}
              y={pad.top}
              width={band}
              height={height - pad.top - pad.bottom}
              fill={hover === i ? 'rgb(100 116 139 / 0.08)' : 'transparent'}
            />
            {#each series as s, j (s.name)}
              {@const value = s.values[i] ?? 0}
              {#if stacked}
                {@const base = stackBase(i, j)}
                {@const part = Math.max(0, value)}
                {#if part > 0}
                  <!-- A 2px surface gap parts a segment from the one below it. -->
                  <rect
                    x={center - barWidth / 2 + 1}
                    y={y(base + part)}
                    width={Math.max(barWidth - 2, 1)}
                    height={Math.max(y(base) - y(base + part) - (base > 0 ? 2 : 0), 1)}
                    rx="3"
                    fill={s.color}
                  >
                    <title>{s.name}: {mask.revealed ? formatCompact(value) : MASK}</title>
                  </rect>
                {/if}
              {:else}
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
              {/if}
            {/each}
            <text x={center} y={height - 6} text-anchor="middle" class="fill-slate-500 text-[11px]">{text}</text>
          </g>
        {/each}
      </svg>
      <MaskedAxis {width} {height} {pad} />
    </div>
    <!-- The legend doubles as the readout: hovering a column shows its values. -->
    <div class="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-xs" data-testid="bar-legend">
      {#if hover !== null}<span class="muted tabular">{labels[hover]}</span>{/if}
      {#each series as s (s.name)}
        <span class="inline-flex items-center gap-1.5">
          <span class="size-2.5 rounded-sm" style:background={s.color}></span>
          <span class="muted">{s.name}</span>
          {#if hover !== null}
            <span class="tabular font-medium"><Masked text={formatCompact(s.values[hover] ?? 0)} /></span>
          {/if}
        </span>
      {/each}
    </div>
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">No data yet.</p>
{/if}
