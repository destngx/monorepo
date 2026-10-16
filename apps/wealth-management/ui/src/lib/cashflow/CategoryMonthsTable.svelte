<script lang="ts">
  import type { CategoryMonths } from '$lib/api/types';
  import { formatCompact, formatMonth, formatVND, MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { columnTotals } from './yearly';

  interface Props {
    year: number;
    rows: CategoryMonths[];
    /** Months after this many have not happened yet and show as blank. */
    monthsElapsed: number;
  }

  let { year, rows, monthsElapsed }: Props = $props();

  const months = $derived(
    Array.from({ length: 12 }, (_, i) => formatMonth(`${year}-${String(i + 1).padStart(2, '0')}`, true)),
  );
  const totals = $derived(columnTotals(rows));
  const grandTotal = $derived(rows.reduce((sum, row) => sum + row.total, 0));

  // Compact figures keep twelve columns readable; the full amount is in the title.
  const cell = (value: number) => (mask.revealed ? (value ? formatCompact(value) : '-') : MASK);
  const full = (value: number) => (mask.revealed ? formatVND(value) : undefined);
</script>

{#if rows.length}
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[56rem] text-sm">
      <thead>
        <tr class="muted border-b border-slate-900/10 text-xs uppercase dark:border-white/10">
          <th class="px-5 py-2 text-left font-medium">Category</th>
          {#each months as month, i (i)}
            <th class="px-2 py-2 text-right font-medium {i >= monthsElapsed ? 'opacity-50' : ''}">{month}</th>
          {/each}
          <th class="px-5 py-2 text-right font-medium">Total</th>
          <th class="px-5 py-2 text-right font-medium">Share</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each rows as row (row.category)}
          <tr>
            <td class="max-w-48 truncate px-5 py-2" title={row.category}>{row.category}</td>
            {#each row.values as value, i (i)}
              <td class="tabular px-2 py-2 text-right {value ? '' : 'muted'}" title={full(value)}>{cell(value)}</td>
            {/each}
            <td class="tabular px-5 py-2 text-right font-medium" title={full(row.total)}>{cell(row.total)}</td>
            <td class="tabular muted px-5 py-2 text-right">
              {grandTotal ? `${((row.total / grandTotal) * 100).toFixed(1)}%` : '-'}
            </td>
          </tr>
        {/each}
      </tbody>
      <tfoot>
        <tr class="border-t border-slate-900/10 font-medium dark:border-white/10">
          <td class="px-5 py-2">Total</td>
          {#each totals as value, i (i)}
            <td class="tabular px-2 py-2 text-right" title={full(value)}>{cell(value)}</td>
          {/each}
          <td class="tabular px-5 py-2 text-right" title={full(grandTotal)}>{cell(grandTotal)}</td>
          <td class="px-5 py-2"></td>
        </tr>
      </tfoot>
    </table>
  </div>
{:else}
  <p class="muted py-8 text-center text-sm">Nothing recorded for {year}.</p>
{/if}
