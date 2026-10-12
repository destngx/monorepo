<script lang="ts">
  import type { DisplayCurrency, LiquidityTier } from '$lib/accounts';
  import Donut from '$lib/charts/Donut.svelte';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatPercent } from '$lib/format';

  // Cross-currency view (WM-ACC-8): figures arrive already converted to the display currency.
  interface Props {
    display: DisplayCurrency;
    allocation: { label: string; value: number }[];
    tiers: { tier: LiquidityTier; percent: number; shown: number | null }[];
    history: { date: string; value: number }[];
  }

  let { display, allocation, tiers, history }: Props = $props();
</script>

<div class="mt-4 grid gap-4 lg:grid-cols-3">
  <Card title="Allocation by group">
    {#if allocation.length}
      <div class="flex flex-wrap items-center gap-5">
        <Donut items={allocation} size={140} label="Allocation by account group" />
        <ul class="min-w-0 flex-1 space-y-1 text-sm">
          {#each allocation as slice (slice.label)}
            <li class="flex justify-between gap-3">
              <span class="truncate">{slice.label}</span><Money value={slice.value} currency={display} />
            </li>
          {/each}
        </ul>
      </div>
    {:else}
      <p class="muted text-sm">No positive balances to allocate.</p>
    {/if}
  </Card>
  <Card title="Liquidity" subtitle="Immediate: spendable now · T+2: a few days · Illiquid: long holding">
    <ul class="space-y-3">
      {#each tiers as tier (tier.tier)}
        <li>
          <div class="mb-1 flex justify-between gap-3 text-sm">
            <span>{tier.tier}</span>
            <span class="muted tabular whitespace-nowrap"
              ><Money value={tier.shown} currency={display} /> · {formatPercent(tier.percent, 0)}</span
            >
          </div>
          <ProgressBar percent={tier.percent} color="bg-indigo-500" label="{tier.tier} share" />
        </li>
      {/each}
    </ul>
  </Card>
  <Card title="Balance history" subtitle="All accounts, last 12 months at today's rates">
    <LineChart points={history} height={180} label="12-month total balance in {display}" />
  </Card>
</div>
