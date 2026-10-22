<script lang="ts">
  import type { Confluence, Timeframe } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Card, Icon } from '@libs/ui-kit';
  import { TIMEFRAME_LABELS, formatNumber, formatPrice, pairTone, rsiTone, trendTone } from '../market';
  import SourceBadge from '$lib/components/SourceBadge.svelte';

  interface Props {
    confluence: Confluence;
    /** The timeframe currently on screen; its row is highlighted. */
    timeframe: Timeframe;
  }

  let { confluence, timeframe }: Props = $props();

  const timeframeOnly = (tf: string) => TIMEFRAME_LABELS[tf as Timeframe] ?? tf;
</script>

<Card title="Multi-timeframe confluence" subtitle="How the timeframes agree; higher timeframes weigh more" class="mt-4">
  {#snippet actions()}
    <Badge tone={trendTone[confluence.bias]}>Bias: {confluence.bias}</Badge>
    <Badge tone="blue">Score {confluence.score}/10</Badge>
  {/snippet}
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[40rem] text-sm">
      <thead>
        <tr class="muted text-left text-xs uppercase">
          <th class="px-5 py-2">Timeframe</th>
          <th class="px-5 py-2">Trend</th>
          <th class="px-5 py-2">Phase</th>
          <th class="px-5 py-2 text-right">RSI</th>
          <th class="px-5 py-2 text-right">Price</th>
          <th class="px-5 py-2">Source</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
        {#each confluence.views as view (view.timeframe)}
          <tr class={view.error ? 'opacity-60' : view.timeframe === timeframe ? 'bg-accent-500/5' : ''}>
            <td class="px-5 py-2 font-medium">{timeframeOnly(view.timeframe)}</td>
            {#if view.error}
              <td class="px-5 py-2 text-xs text-rose-700 dark:text-rose-300" colspan="4">
                <span class="inline-flex items-start gap-1"
                  ><Icon name="alert" size={14} class="mt-0.5" /> {view.error}</span
                >
              </td>
            {:else}
              <td class="px-5 py-2"><Badge tone={trendTone[view.trend]}>{view.trend}</Badge></td>
              <td class="px-5 py-2">{view.phase}</td>
              <td class="tabular px-5 py-2 text-right">
                {#if view.rsi !== null}
                  <span
                    class={rsiTone(view.rsi) === 'red'
                      ? 'text-rose-600 dark:text-rose-400'
                      : rsiTone(view.rsi) === 'green'
                        ? 'text-emerald-600 dark:text-emerald-400'
                        : ''}>{formatNumber(view.rsi, 1)}</span
                  >
                {:else}<span class="muted">-</span>{/if}
              </td>
              <td class="tabular px-5 py-2 text-right">{formatPrice(view.price)}</td>
            {/if}
            <td class="px-5 py-2"><SourceBadge source={view.source} /></td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  {#if confluence.pairs.length}
    <div class="mt-3 flex flex-wrap gap-2 text-sm">
      {#each confluence.pairs as pair (`${pair.higher}-${pair.lower}`)}
        <span class="inline-flex items-center gap-1.5">
          <span class="muted text-xs">{timeframeOnly(pair.higher)} vs {timeframeOnly(pair.lower)}</span>
          <Badge tone={pairTone[pair.status]}>{pair.status}</Badge>
        </span>
      {/each}
    </div>
  {/if}
</Card>
