<script lang="ts">
  import type { Technicals } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Card } from '@libs/ui-kit';
  import { formatDate } from '$lib/format';
  import { formatPrice } from '../market';

  interface Props {
    technicals: Technicals;
  }

  let { technicals }: Props = $props();

  const levels = $derived.by(() => {
    const { support, resistance } = technicals.supportResistance;
    // Nearest first: support below price descending, resistance above ascending.
    return { support: [...support].sort((a, b) => b - a), resistance: [...resistance].sort((a, b) => a - b) };
  });
</script>

<div class="space-y-4">
  <Card title="Support and resistance" subtitle="Nearest levels first">
    <div class="grid grid-cols-2 gap-4 text-sm">
      <div>
        <p class="mb-1 text-xs font-semibold text-rose-700 uppercase dark:text-rose-300">Resistance</p>
        {#if levels.resistance.length}
          <ul class="tabular space-y-0.5">
            {#each levels.resistance as level, i (i)}<li>{formatPrice(level)}</li>{/each}
          </ul>
        {:else}<p class="muted text-xs">None found</p>{/if}
      </div>
      <div>
        <p class="mb-1 text-xs font-semibold text-emerald-700 uppercase dark:text-emerald-300">Support</p>
        {#if levels.support.length}
          <ul class="tabular space-y-0.5">
            {#each levels.support as level, i (i)}<li>{formatPrice(level)}</li>{/each}
          </ul>
        {:else}<p class="muted text-xs">None found</p>{/if}
      </div>
    </div>
  </Card>

  <Card title="Open fair value gaps" subtitle="Price imbalances not yet filled">
    {#if technicals.fairValueGaps.length}
      <ul class="space-y-1.5 text-sm">
        {#each technicals.fairValueGaps as gap, i (i)}
          <li class="flex flex-wrap items-center justify-between gap-2">
            <Badge tone={gap.type === 'BULLISH' ? 'green' : 'red'}>{gap.type}</Badge>
            <span class="tabular">{formatPrice(gap.bottom)} - {formatPrice(gap.top)}</span>
            <span class="muted text-xs">{formatDate(gap.time)}</span>
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No open gaps.</p>
    {/if}
  </Card>
</div>
