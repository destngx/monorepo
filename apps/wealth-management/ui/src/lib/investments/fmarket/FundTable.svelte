<script lang="ts">
  import { base } from '$app/paths';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import { formatDate, formatSignedPercent } from '$lib/format';
  import { sortFunds, type FundSort, type FundSummary } from '../fmarket';

  let { funds, label }: { funds: FundSummary[]; label: string } = $props();

  let sortKey = $state<FundSort>('return12m');
  const rows = $derived(sortFunds(funds, sortKey));

  const nav = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 2 });
  const tone = (value: number | null) =>
    value === null
      ? 'muted'
      : value >= 0
        ? 'text-emerald-600 dark:text-emerald-400'
        : 'text-orange-600 dark:text-orange-400';

  const columns: { key: FundSort; label: string; align: string }[] = [
    { key: 'code', label: 'Fund', align: 'text-left' },
    { key: 'nav', label: 'NAV (₫)', align: 'text-right' },
    { key: 'return12m', label: '12 months', align: 'text-right' },
    { key: 'annualized36m', label: '3Y annualised', align: 'text-right' },
  ];
</script>

{#if rows.length}
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[34rem] text-sm">
      <thead>
        <tr class="muted text-xs uppercase">
          {#each columns as column (column.key)}
            <th class="px-5 py-2 {column.align}" aria-sort={sortKey === column.key ? 'descending' : 'none'}>
              <button
                type="button"
                class="rounded uppercase hover:text-slate-900 dark:hover:text-white {sortKey === column.key
                  ? 'text-indigo-600 dark:text-indigo-300'
                  : ''}"
                onclick={() => (sortKey = column.key)}>{column.label}</button
              >
            </th>
          {/each}
          <th class="px-5 py-2 text-right">Updated</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each rows as fund (fund.id)}
          <!-- The whole row opens the fund: the link covers it (WM-UI-18). -->
          <tr class="row-link relative">
            <td class="px-5 py-2">
              <a
                href="{base}/investments/funds/{encodeURIComponent(fund.code)}"
                class="group flex items-center gap-3 after:absolute after:inset-0 after:content-['']"
              >
                {#if fund.issuerLogo}
                  <img
                    src={fund.issuerLogo}
                    alt=""
                    class="size-8 shrink-0 rounded-lg bg-white object-contain p-0.5"
                    loading="lazy"
                  />
                {:else}
                  <span class="size-8 shrink-0 rounded-lg bg-slate-900/5 dark:bg-white/10"></span>
                {/if}
                <span class="min-w-0">
                  <span
                    class="block font-medium text-indigo-600 underline-offset-2 group-hover:underline dark:text-indigo-300"
                    >{fund.code}</span
                  >
                  <span class="muted block max-w-[9rem] truncate sm:max-w-[22rem] text-xs" title={fund.name}
                    >{fund.issuer ? `${fund.issuer} · ` : ''}{fund.name}</span
                  >
                </span>
              </a>
            </td>
            <td class="tabular px-5 py-2 text-right">{fund.nav === null ? '-' : nav.format(fund.nav)}</td>
            <td class="tabular px-5 py-2 text-right font-medium {tone(fund.return12m)}"
              >{formatSignedPercent(fund.return12m, 2)}</td
            >
            <td class="tabular px-5 py-2 text-right font-medium {tone(fund.annualized36m)}"
              >{formatSignedPercent(fund.annualized36m, 2)}</td
            >
            <td class="muted tabular px-5 py-2 text-right text-xs">{formatDate(fund.updatedAt)}</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{:else}
  <EmptyState title="No {label.toLowerCase()} funds listed" detail="fmarket returned no active funds of this type." />
{/if}
