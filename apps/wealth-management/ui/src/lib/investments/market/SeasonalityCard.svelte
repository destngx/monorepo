<script lang="ts">
  import type { SeasonalityBucket } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Card } from '@libs/ui-kit';
  import { formatPercent, formatSignedPercent } from '$lib/format';
  import { changeClass, formatNumber, groupSeasonality, rankMark } from '../market';

  interface Props {
    buckets: SeasonalityBucket[];
  }

  let { buckets }: Props = $props();

  const seasonality = $derived(groupSeasonality(buckets));
</script>

<Card title="Seasonality" subtitle="Daily returns grouped by calendar bucket; rank 1 is the best" class="mt-4">
  {#if seasonality.length}
    <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
      {#each seasonality as group (group.type)}
        <div class="min-w-0">
          <h3 class="mb-2 text-xs font-semibold tracking-wide text-stone-600 uppercase dark:text-stone-400">
            {group.title}
          </h3>
          <div class="overflow-x-auto">
            <table class="w-full min-w-[26rem] text-xs">
              <thead>
                <tr class="muted text-left uppercase">
                  <th class="py-1.5 pr-2">Bucket</th>
                  <th class="px-2 py-1.5 text-right">Avg</th>
                  <th class="px-2 py-1.5 text-right">Win</th>
                  <th class="px-2 py-1.5 text-right" title="Profit factor">PF</th>
                  <th class="px-2 py-1.5 text-right" title="Standard deviation">SD</th>
                  <th class="px-2 py-1.5 text-right">Score</th>
                  <th class="px-2 py-1.5 text-right">N</th>
                  <th class="py-1.5 pl-2 text-right">Rank</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
                {#each group.rows as row (row.key)}
                  {@const mark = rankMark(row.rank, group.worstRank)}
                  <tr class={mark === 'best' ? 'bg-emerald-500/10' : mark === 'worst' ? 'bg-rose-500/10' : ''}>
                    <td class="py-1.5 pr-2 font-medium">{row.label}</td>
                    <td class="tabular px-2 py-1.5 text-right {changeClass(row.avgReturn)}"
                      >{formatSignedPercent(row.avgReturn, 2)}</td
                    >
                    <td class="tabular px-2 py-1.5 text-right">{formatPercent(row.winRate, 0)}</td>
                    <td class="tabular px-2 py-1.5 text-right">{formatNumber(row.profitFactor)}</td>
                    <td class="tabular px-2 py-1.5 text-right">{formatPercent(row.stdDev, 2)}</td>
                    <td class="tabular px-2 py-1.5 text-right">{formatNumber(row.score)}</td>
                    <td class="tabular px-2 py-1.5 text-right">{row.n}</td>
                    <td class="tabular py-1.5 pl-2 text-right">
                      {#if mark === 'best'}<Badge tone="green">Best</Badge>{:else if mark === 'worst'}<Badge tone="red"
                          >Worst</Badge
                        >{:else}{row.rank}{/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/each}
    </div>
  {:else}
    <p class="muted text-sm">Not enough daily history to group returns.</p>
  {/if}
</Card>
