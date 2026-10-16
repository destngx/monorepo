<script lang="ts">
  import type { LiquidityTier } from '$lib/accounts';
  import type { Allocation } from '$lib/api/types';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatPercent } from '$lib/format';
  import AllocationSummary from '$lib/investments/AllocationSummary.svelte';
  import type { Section } from '$lib/investments/fmarket';
  import SectionError from '$lib/investments/fmarket/SectionError.svelte';

  // WM-ACC-8: every figure is VND (D25). The allocation is the engine's one
  // allocation (ROADMAP C4), loaded on its own so a failure leaves the rest.
  interface Props {
    allocation: Section<Allocation>;
    tiers: { tier: LiquidityTier; percent: number; value: number }[];
    history: { date: string; value: number }[];
  }

  let { allocation, tiers, history }: Props = $props();
</script>

<div class="mt-4 grid gap-4 lg:grid-cols-3">
  {#if allocation.ok}
    <AllocationSummary allocation={allocation.value} />
  {:else}
    <Card title="Asset allocation">
      <SectionError title="The allocation is unavailable" message={allocation.error} />
    </Card>
  {/if}
  <Card title="Liquidity" subtitle="Immediate: spendable now · T+2: a few days · Illiquid: long holding">
    <ul class="space-y-3">
      {#each tiers as tier (tier.tier)}
        <li>
          <div class="mb-1 flex justify-between gap-3 text-sm">
            <span>{tier.tier}</span>
            <span class="muted tabular whitespace-nowrap"
              ><Money value={tier.value} /> · {formatPercent(tier.percent, 0)}</span
            >
          </div>
          <ProgressBar percent={tier.percent} color="bg-indigo-500" label="{tier.tier} share" />
        </li>
      {/each}
    </ul>
  </Card>
  <Card title="Balance history" subtitle="All accounts, last 12 months">
    <LineChart points={history} height={180} label="12-month total balance in VND" />
  </Card>
</div>
