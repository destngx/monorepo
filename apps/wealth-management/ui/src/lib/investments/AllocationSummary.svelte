<script lang="ts">
  import { base } from '$app/paths';
  import type { Allocation } from '$lib/api/types';
  import Donut from '$lib/charts/Donut.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import { Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatPercent } from '$lib/format';
  import { ALLOCATION_COLORS, allocationSlices, allocationStatus } from './allocation';

  // Compact view of the one engine allocation (ROADMAP C4); the full view
  // with targets and drift lives on Investments.
  let { allocation, class: className = '' }: { allocation: Allocation; class?: string } = $props();

  const slices = $derived(allocationSlices(allocation));
  const status = $derived(allocationStatus(allocation));
  const lines = $derived(allocation.lines.filter((line) => line.value > 0));
</script>

<Card title="Asset allocation" subtitle="Share of everything you own, money lent included" class={className}>
  {#snippet actions()}
    <Badge status={status.tone}>{status.label}</Badge>
  {/snippet}
  {#if slices.length}
    <div class="flex flex-wrap items-center gap-5">
      <Donut items={slices} size={140} label="Asset allocation" />
      <ul class="min-w-0 flex-1 space-y-1 text-sm">
        {#each lines as line (line.class)}
          <li class="flex items-center justify-between gap-3">
            <span class="flex min-w-0 items-center gap-2">
              <span class="size-2.5 shrink-0 rounded-full" style:background-color={ALLOCATION_COLORS[line.class]}
              ></span>
              <span class="truncate">{line.label}</span>
            </span>
            <span class="muted tabular whitespace-nowrap"
              ><Money value={line.value} /> · {formatPercent(line.share, 0)}</span
            >
          </li>
        {/each}
      </ul>
    </div>
  {:else}
    <p class="muted text-sm">No positive balances to allocate.</p>
  {/if}
  <a href="{base}/investments" class="mt-3 inline-block text-xs text-accent-600 hover:underline dark:text-accent-400"
    >Targets and drift on Investments</a
  >
</Card>
