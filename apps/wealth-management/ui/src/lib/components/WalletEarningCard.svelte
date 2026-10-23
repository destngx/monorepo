<script lang="ts">
  import { track, Card, Deferred, SkeletonTable } from '@libs/ui-kit';
  import type { Streamed } from '$lib/api/load';
  import type { FlexibleSavings } from '$lib/api/types';
  import { rateLabel } from '$lib/bills/idleCash';
  import { STATUS } from '$lib/status';
  import Badge from './Badge.svelte';
  import SourceBadge from './SourceBadge.svelte';

  // The live rates of e-wallet earning balances (D41), e.g. ZaloPay Số dư sinh
  // lời and MoMo Túi Thần Tài, as the market data server scrapes them.
  let { data, class: className = '' }: { data: Streamed<FlexibleSavings>; class?: string } = $props();
  const savings = track(() => data);
</script>

<Card
  title="Wallet earning balances"
  subtitle="Live rates of e-wallet balances you can spend or withdraw any time, % per year"
  class={className}
>
  {#snippet actions()}
    {#if savings.value?.source}<SourceBadge source={savings.value.source} />{/if}
  {/snippet}
  <Deferred {data} label="wallet earning balances">
    {#snippet children(loaded)}
      <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5" data-testid="flexible-savings">
        {#each loaded.products as product (product.id)}
          <li class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 py-1.5">
            <span class="min-w-0">
              {product.provider}
              {product.product}
              {#if product.url}
                <a class="link text-xs" href={product.url} target="_blank" rel="noopener noreferrer"
                  >Open<span class="sr-only"> {product.provider} {product.product} (new tab)</span></a
                >
              {/if}
            </span>
            <span class="flex items-center gap-1.5">
              {#if product.backing === 'business_cooperation'}<Badge status="warning">not a bank deposit</Badge>{/if}
              {#if product.stale}<Badge status="warning">stale</Badge>{/if}
              <span class="tabular font-medium">{rateLabel(product.rate)}</span>
            </span>
          </li>
        {:else}
          <li class="muted py-1.5">No wallet earning balance rate is published right now.</li>
        {/each}
      </ul>
      {#if loaded.failed.length}
        <p class="mt-1 text-xs {STATUS.warning.ink}">Could not load: {loaded.failed.join(', ')}.</p>
      {/if}
    {/snippet}
    {#snippet skeleton()}<SkeletonTable rows={2} columns={2} label="wallet earning balances" />{/snippet}
  </Deferred>
</Card>
