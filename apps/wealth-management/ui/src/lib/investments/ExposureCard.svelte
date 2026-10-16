<script lang="ts">
  import type { Exposure } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatPercent, formatSignedPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { ALLOCATION_COLORS } from './allocation';
  import { coverage } from './exposure';

  // ROADMAP M1: the headline answer first (how much of the money moves with markets and how much
  // it moved today), then the classes of the one allocation.
  let { exposure }: { exposure: Exposure } = $props();

  const covers = $derived(coverage(exposure));
  const shown = $derived(exposure.classes.filter((c) => c.value > 0));
</script>

<Card title="My exposure" subtitle="What you hold and how much of it moves with markets">
  <div class="border-l-4 pl-3 {STATUS.info.edge}">
    <p class="text-lg" data-testid="exposure-headline">
      <strong class="tabular">{formatPercent(exposure.volatileShare, 0)}</strong> of your assets move with markets;{' '}{#if exposure.change !== null}
        today they moved <strong><Money value={exposure.change} signed tone /></strong>
        ({formatSignedPercent(exposure.changePct, 2)}).
      {:else}
        today's change is not available.
      {/if}
    </p>
    <p class="muted mt-1 text-xs">
      Fund certificates and crypto move with markets: <Money value={exposure.volatileValue} /> of
      <Money value={exposure.total} />. Coins change over about 24 hours, funds since their previous NAV.
      {#if covers === 'partial'}
        Today's change covers <Money value={exposure.covered} /> of it; the rest has no price move (see notes).
      {/if}
    </p>
  </div>

  {#if shown.length}
    <div class="mt-4 flex h-3 w-full overflow-hidden rounded-full" role="img" aria-label="Share of each asset class">
      {#each shown as c (c.class)}
        <span style:width="{c.share}%" style:background-color={ALLOCATION_COLORS[c.class]} title="{c.label} {c.share}%"
        ></span>
      {/each}
    </div>
    <div class="-mx-5 mt-3 overflow-x-auto">
      <table class="w-full min-w-[36rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Class</th>
            <th class="px-5 py-2 text-right">Value</th>
            <th class="px-5 py-2 text-right">Share</th>
            <th class="px-5 py-2 text-right">Today</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each exposure.classes as c (c.class)}
            <tr>
              <td class="px-5 py-2">
                <span class="flex items-center gap-2">
                  <span class="size-2.5 shrink-0 rounded-full" style:background-color={ALLOCATION_COLORS[c.class]}
                  ></span>
                  {c.label}
                  <Badge tone={c.volatile ? 'blue' : 'neutral'}>{c.volatile ? 'Moves with markets' : 'Fixed VND'}</Badge
                  >
                </span>
              </td>
              <td class="px-5 py-2 text-right"><Money value={c.value} /></td>
              <td class="tabular px-5 py-2 text-right">{formatPercent(c.share, 1)}</td>
              <td class="px-5 py-2 text-right">
                {#if c.change !== null}
                  <Money value={c.change} signed tone />
                  <span class="muted block text-xs">{formatSignedPercent(c.changePct, 2)}</span>
                {:else}<span class="muted">-</span>{/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <p class="muted mt-4 text-sm">No positive balances or valued holdings yet.</p>
  {/if}
</Card>
