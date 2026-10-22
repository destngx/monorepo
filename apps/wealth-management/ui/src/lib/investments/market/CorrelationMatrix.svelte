<script lang="ts">
  import type { CorrelationMatrix } from '$lib/api/types';
  import { correlationStyle, formatNumber } from '../market';

  // Pearson correlation of returns; null cells share too few bars (shown as n/a).
  let { matrix }: { matrix: CorrelationMatrix } = $props();
</script>

{#if matrix.symbols.length > 1}
  <div class="-mx-5 overflow-x-auto px-5">
    <table class="mx-auto text-xs">
      <thead>
        <tr>
          <th class="p-1"><span class="sr-only">Symbol</span></th>
          {#each matrix.symbols as symbol (symbol)}
            <th class="muted p-1 text-center font-medium" scope="col">{symbol}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each matrix.symbols as row, i (row)}
          <tr>
            <th class="muted p-1 pr-2 text-right font-medium" scope="row">{row}</th>
            {#each matrix.symbols as column, j (column)}
              {@const value = matrix.values[i]?.[j] ?? null}
              <td
                class="tabular h-9 min-w-12 rounded-md border border-white/40 text-center dark:border-white/5 {value ===
                null
                  ? 'muted bg-stone-900/5 dark:bg-white/5'
                  : 'text-stone-900 dark:text-stone-50'}"
                style={correlationStyle(value)}
                title="{row} / {column}: {value === null ? 'not enough shared bars' : formatNumber(value)}"
                >{value === null ? 'n/a' : formatNumber(value)}</td
              >
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  <p class="muted mt-3 flex flex-wrap items-center gap-3 text-xs">
    <span class="inline-flex items-center gap-1"
      ><span class="inline-block size-3 rounded" style={correlationStyle(-1)}></span> Moves opposite</span
    >
    <span class="inline-flex items-center gap-1"
      ><span class="inline-block size-3 rounded bg-stone-900/5 dark:bg-white/5"></span> Unrelated / n/a</span
    >
    <span class="inline-flex items-center gap-1"
      ><span class="inline-block size-3 rounded" style={correlationStyle(1)}></span> Moves together</span
    >
  </p>
{:else}
  <p class="muted text-sm">Not enough series to correlate.</p>
{/if}
