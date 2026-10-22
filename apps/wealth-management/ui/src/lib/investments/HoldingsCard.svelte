<script lang="ts">
  import type { ValuedHolding } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Money from '$lib/components/Money.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { formatDate } from '$lib/format';
  import { holdingSource, quantity } from './holdings';

  interface Props {
    title: string;
    rows: ValuedHolding[];
    /** Section total in VND. */
    total: number;
  }

  let { title, rows, total }: Props = $props();
</script>

<Card {title} subtitle="{rows.length} holdings">
  {#if rows.length}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[52rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Platform</th>
            <th class="px-5 py-2">Symbol</th>
            <th class="px-5 py-2 text-right">Quantity</th>
            <th class="px-5 py-2 text-right">Sheet price</th>
            <th class="px-5 py-2 text-right">Live price</th>
            <th class="px-5 py-2 text-right">Value</th>
            <th class="px-5 py-2 text-right">VND</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
          {#each rows as holding, i (i)}
            {@const label = holdingSource(holding)}
            <tr>
              <td class="px-5 py-2">{holding.platform}</td>
              <td class="px-5 py-2 font-medium">{holding.symbol || '-'}</td>
              <td class="tabular px-5 py-2 text-right">{quantity(holding.quantity)}</td>
              <td class="px-5 py-2 text-right"
                >{#if holding.price !== null}<Money value={holding.price} currency={holding.currency} />{:else}<span
                    class="muted">-</span
                  >{/if}</td
              >
              <td class="px-5 py-2 text-right">
                {#if holding.livePrice}
                  <Money value={holding.livePrice.price} currency={holding.livePrice.currency} />
                  <span class="block"
                    ><SourceBadge
                      source={holding.livePrice.source ?? { provider: holding.livePrice.provider }}
                      detail={holding.livePrice.timestamp ? formatDate(holding.livePrice.timestamp) : undefined}
                    /></span
                  >
                {:else}<span class="muted">-</span>{/if}
              </td>
              <td class="px-5 py-2 text-right">
                {#if holding.value !== null}
                  <Money value={holding.value} currency={holding.currency} />
                  {#if label}<Badge tone={holding.valueSource === 'quantity_x_live_price' ? 'blue' : 'neutral'}
                      >{label}</Badge
                    >{/if}
                {:else}<span class="muted">No value</span>{/if}
              </td>
              <td class="px-5 py-2 text-right">
                {#if holding.valueVND !== null}<Money value={holding.valueVND} />{:else}
                  <span class="text-xs text-amber-700 dark:text-amber-300">{holding.issue || 'Not valued'}</span>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
        <tfoot>
          <tr class="border-t border-stone-900/10 font-semibold dark:border-white/10">
            <td class="px-5 py-2" colspan="6">Total</td>
            <td class="px-5 py-2 text-right"><Money value={total} /></td>
          </tr>
        </tfoot>
      </table>
    </div>
  {:else}
    <EmptyState title="No {title.toLowerCase()} holdings" />
  {/if}
</Card>
