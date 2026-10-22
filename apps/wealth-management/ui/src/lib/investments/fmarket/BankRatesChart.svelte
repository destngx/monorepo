<script lang="ts">
  import type { BankRate } from '$lib/api/types';
  import { EmptyState } from '@libs/ui-kit';
  import { formatPercent } from '$lib/format';

  let { rates }: { rates: BankRate[] } = $props();

  // Horizontal bars read better than columns once bank names get long.
  const rows = $derived([...rates].filter((rate) => rate.rate > 0).sort((a, b) => b.rate - a.rate));
  const max = $derived(Math.max(...rows.map((rate) => rate.rate), 0));
</script>

{#if rows.length}
  <ul class="space-y-2" aria-label="Savings rate by bank">
    {#each rows as rate, i (i)}
      <li class="grid grid-cols-[7.5rem_1fr_3.5rem] items-center gap-3 text-sm sm:grid-cols-[10rem_1fr_4rem]">
        <span class="truncate" title={rate.bank}>{rate.bank}{rate.tenor ? ` · ${rate.tenor}` : ''}</span>
        <span class="h-3 overflow-hidden rounded-sm bg-stone-900/5 dark:bg-white/5">
          <span
            class="block h-full rounded-sm {i === 0 ? 'bg-emerald-500' : 'bg-accent-500/80'}"
            style:width="{max ? (rate.rate / max) * 100 : 0}%"
          ></span>
        </span>
        <span class="tabular text-right font-medium">{formatPercent(rate.rate, 2)}</span>
      </li>
    {/each}
  </ul>
{:else}
  <EmptyState title="No bank rates published" />
{/if}
