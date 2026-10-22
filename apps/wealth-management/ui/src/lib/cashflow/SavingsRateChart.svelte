<script lang="ts">
  import type { CashFlow, SavingsTarget } from '$lib/api/types';
  import { ticks, scale, leavesReadout } from '@libs/ui-kit';
  import { PRIMARY } from '$lib/charts/palette';
  import Masked from '$lib/components/Masked.svelte';
  import { formatCompact, formatMonth, formatPercent } from '$lib/format';
  import { SAVINGS_RATE_LIMIT, savingsRateDomain } from './yearly';

  interface Props {
    months: CashFlow[];
    monthsElapsed: number;
    target: SavingsTarget;
    height?: number;
  }

  // WM-DASH-8: each month's savings rate against the healthy band and the target.
  let { months, monthsElapsed, target, height = 200 }: Props = $props();

  const BAR = PRIMARY;
  const width = 640;
  const pad = { top: 12, right: 8, bottom: 24, left: 40 };

  // A month without income has no rate; it gets no bar rather than a misleading 0%.
  const rates = $derived(months.map((m, i) => (i < monthsElapsed && m.income > 0 ? m.savingsRate : null)));
  const domain = $derived(
    savingsRateDomain(
      rates.filter((r) => r !== null),
      target.target,
    ),
  );
  const yTicks = $derived(ticks(domain));
  const y = $derived(scale(domain, [height - pad.bottom, pad.top]));
  const clamped = (rate: number) => Math.max(domain[0], Math.min(domain[1], rate));
  const band = $derived((width - pad.left - pad.right) / Math.max(months.length, 1));
  const barWidth = $derived(Math.min(24, band * 0.6));

  let hover = $state<number | null>(null);
</script>

<div role="presentation" onpointerleave={(event) => leavesReadout(event) && (hover = null)}>
  <svg
    viewBox="0 0 {width} {height}"
    class="h-auto w-full"
    role="img"
    aria-label="Savings rate per month, with the {target.healthy}-{target.target}% healthy band"
  >
    <rect
      x={pad.left}
      width={width - pad.left - pad.right}
      y={y(target.target)}
      height={y(target.healthy) - y(target.target)}
      class="fill-emerald-500/10"
    />
    {#each yTicks as tick (tick)}
      <line
        x1={pad.left}
        x2={width - pad.right}
        y1={y(tick)}
        y2={y(tick)}
        class={tick === 0 ? 'stroke-stone-900/30 dark:stroke-white/30' : 'stroke-stone-900/10 dark:stroke-white/10'}
      />
      <text x={pad.left - 8} y={y(tick)} dy="0.32em" text-anchor="end" class="fill-stone-500 text-[11px]">{tick}%</text>
    {/each}
    <line
      x1={pad.left}
      x2={width - pad.right}
      y1={y(target.target)}
      y2={y(target.target)}
      stroke-dasharray="4 4"
      class="stroke-emerald-600 dark:stroke-emerald-400"
    />
    {#each months as month, i (month.month)}
      {@const rate = rates[i]}
      {@const center = pad.left + band * i + band / 2}
      <g role="presentation" onpointerenter={() => (hover = i)}>
        <rect
          x={pad.left + band * i}
          y={pad.top}
          width={band}
          height={height - pad.top - pad.bottom}
          fill={hover === i ? 'rgb(100 116 139 / 0.08)' : 'transparent'}
        />
        {#if rate !== null}
          <rect
            x={center - barWidth / 2}
            y={Math.min(y(clamped(rate)), y(0))}
            width={barWidth}
            height={Math.max(Math.abs(y(clamped(rate)) - y(0)), 1)}
            rx="3"
            fill={BAR}
            fill-opacity={rate < 0 ? 0.45 : 1}
          />
          {#if Math.abs(rate) > SAVINGS_RATE_LIMIT}
            <!-- Clipped at the edge; the readout has the real figure. -->
            <text
              x={center}
              y={rate > 0 ? pad.top - 2 : height - pad.bottom + 10}
              text-anchor="middle"
              class="fill-stone-500 text-[10px]">{rate > 0 ? '▲' : '▼'}</text
            >
          {/if}
        {/if}
        <text
          x={center}
          y={height - 6}
          text-anchor="middle"
          class="text-[11px] {i < monthsElapsed ? 'fill-stone-500' : 'fill-stone-400/50'}"
          >{formatMonth(month.month, true)}</text
        >
      </g>
    {/each}
  </svg>
  <div class="mt-2 flex flex-wrap items-baseline gap-x-4 gap-y-1 text-xs" data-testid="savings-legend">
    {#if hover !== null}
      {@const month = months[hover]}
      <span class="muted tabular">{formatMonth(month.month, true)}</span>
      {#if rates[hover] === null}
        <span class="muted">{hover < monthsElapsed ? 'No income' : 'Not yet'}</span>
      {:else}
        <span class="tabular font-medium">Kept {formatPercent(month.savingsRate)}</span>
        <span class="muted tabular"><Masked text={formatCompact(month.cashFlow)} /> saved</span>
      {/if}
    {:else}
      <span class="inline-flex items-center gap-1.5">
        <span class="size-2.5 rounded-sm" style:background={BAR}></span><span class="muted">Savings rate</span>
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="size-2.5 rounded-sm bg-emerald-500/25"></span><span class="muted"
          >Healthy {target.healthy}-{target.target}%</span
        >
      </span>
      <span class="inline-flex items-center gap-1.5">
        <span class="w-3 border-t-2 border-dashed border-emerald-600 dark:border-emerald-400"></span><span class="muted"
          >Target {target.target}%</span
        >
      </span>
    {/if}
  </div>
</div>
