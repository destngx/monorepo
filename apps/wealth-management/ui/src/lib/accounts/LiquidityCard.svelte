<script lang="ts">
  import type { LiquidityTier } from '$lib/accounts';
  import { Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatPercent } from '$lib/format';

  // WM-ACC-8: balances by how fast they can be spent; every figure is VND (D25).
  let { tiers }: { tiers: { tier: LiquidityTier; percent: number; value: number }[] } = $props();
</script>

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
        <ProgressBar percent={tier.percent} color="bg-accent-500" label="{tier.tier} share" />
      </li>
    {/each}
  </ul>
</Card>
