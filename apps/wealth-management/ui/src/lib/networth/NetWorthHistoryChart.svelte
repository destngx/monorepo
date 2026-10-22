<script lang="ts">
  import type { NetWorthMilestone, NetWorthPoint } from '$lib/api/types';
  import { extent, scale, ticks } from '$lib/charts/geometry';
  import { leavesReadout } from '$lib/charts/hover';
  import MaskedAxis from '$lib/charts/MaskedAxis.svelte';
  import { NET_WORTH } from '$lib/charts/palette';
  import Money from '$lib/components/Money.svelte';
  import { formatCompact, formatMonth } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { lineSegments, sourceLabel } from './history';

  interface Props {
    points: NetWorthPoint[];
    milestones: NetWorthMilestone[];
    height?: number;
  }

  // ROADMAP P3: monthly net worth on one axis. Measured months (snapshot, today) are solid with
  // filled markers; estimated months are dashed with hollow markers; milestones carry a diamond.
  let { points, milestones, height = 240 }: Props = $props();

  const width = 640;
  const pad = { top: 16, right: 12, bottom: 24, left: 56 };

  const yExtent = $derived(extent(points.map((p) => p.netWorth)));
  const yTicks = $derived(ticks(yExtent));
  const y = $derived(scale(yExtent, [height - pad.bottom, pad.top]));
  const band = $derived((width - pad.left - pad.right) / Math.max(points.length, 1));
  const x = (index: number) => pad.left + band * index + band / 2;
  const segments = $derived(lineSegments(points));
  const milestoneMonths = $derived(new Set(milestones.map((m) => m.month)));
  const xLabels = $derived(
    points.length > 2 ? [0, Math.floor((points.length - 1) / 2), points.length - 1] : points.map((_, i) => i),
  );

  let hover = $state<number | null>(null);
  const hovered = $derived(hover === null ? null : points[hover]);
  const hoveredMilestones = $derived(hovered ? milestones.filter((m) => m.month === hovered.month) : []);
</script>

{#if points.length > 1}
  <div role="presentation" onpointerleave={(event) => leavesReadout(event) && (hover = null)}>
    <div class="relative">
      <svg viewBox="0 0 {width} {height}" class="block h-auto w-full" role="img" aria-label="Net worth per month">
        {#each yTicks as tick (tick)}
          <line
            x1={pad.left}
            x2={width - pad.right}
            y1={y(tick)}
            y2={y(tick)}
            class="stroke-stone-900/10 dark:stroke-white/10"
          />
          <text x={pad.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="fill-stone-500 text-[11px]"
            >{mask.revealed ? formatCompact(tick) : '•••'}</text
          >
        {/each}
        {#each xLabels as index (index)}
          <text
            x={x(index)}
            y={height - 6}
            text-anchor={index === 0 ? 'start' : index === points.length - 1 ? 'end' : 'middle'}
            class="fill-stone-500 text-[11px]">{formatMonth(points[index].month)}</text
          >
        {/each}
        {#each segments as segment (segment.from)}
          <line
            x1={x(segment.from)}
            y1={y(points[segment.from].netWorth)}
            x2={x(segment.to)}
            y2={y(points[segment.to].netWorth)}
            stroke={NET_WORTH}
            stroke-width="2"
            stroke-linecap="round"
            stroke-dasharray={segment.dashed ? '5 4' : undefined}
            data-dashed={segment.dashed}
          />
        {/each}
        {#each points as point, i (point.month)}
          <g role="presentation" data-month={point.month} onpointerenter={() => (hover = i)}>
            <rect
              x={pad.left + band * i}
              y={pad.top}
              width={band}
              height={height - pad.top - pad.bottom}
              fill={hover === i ? 'rgb(100 116 139 / 0.08)' : 'transparent'}
            />
            {#if milestoneMonths.has(point.month)}
              <path
                d="M{x(i)},{y(point.netWorth) - 16} l5,5 l-5,5 l-5,-5 z"
                class="fill-stone-700 dark:fill-stone-200"
                data-milestone={point.month}
              />
            {/if}
            <circle
              cx={x(i)}
              cy={y(point.netWorth)}
              r="4"
              stroke={NET_WORTH}
              stroke-width="2"
              fill={point.estimated ? 'transparent' : NET_WORTH}
              class={point.estimated ? 'fill-white dark:fill-stone-900' : ''}
              data-estimated={point.estimated}
            />
          </g>
        {/each}
      </svg>
      <MaskedAxis {width} {height} {pad} />
    </div>
    <div class="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-xs" data-testid="networth-readout">
      {#if hovered}
        <span class="muted tabular">{formatMonth(hovered.month)}</span>
        <span class="font-medium tabular"><Money value={hovered.netWorth} /></span>
        <span class="muted">{sourceLabel(hovered)}</span>
        {#if hovered.change !== null}
          <span class="tabular">Change <Money value={hovered.change} /></span>
        {/if}
        {#if hovered.savings !== null && hovered.market !== null}
          <span class="muted tabular"
            >Savings <Money value={hovered.savings} /> · Market and other <Money value={hovered.market} /></span
          >
        {/if}
        {#each hoveredMilestones as milestone (milestone.kind)}
          <span class="font-medium"
            >{#if milestone.kind === 'all_time_high'}◆ All-time high{:else}◆ Crossed{#if milestone.threshold !== null}{' '}<Money
                  value={milestone.threshold}
                />{/if}{/if}</span
          >
        {/each}
      {:else}
        <span class="inline-flex items-center gap-1.5">
          <span class="inline-block h-0.5 w-4" style:background={NET_WORTH}></span><span class="muted"
            >Snapshot or today</span
          >
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span class="inline-block w-4 border-t-2 border-dashed" style:border-color={NET_WORTH}></span><span
            class="muted">Estimated from transactions</span
          >
        </span>
        <span class="inline-flex items-center gap-1.5"
          ><span aria-hidden="true">◆</span><span class="muted">Milestone</span></span
        >
      {/if}
    </div>
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">Not enough history to draw a trend yet.</p>
{/if}
