<script lang="ts">
  import type { Transaction } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import CategoryBadge from './CategoryBadge.svelte';
  import type { SortDir, SortKey } from './filter';
  import type { CheckedHandler, RowToggleHandler, SortHandler, TransactionHandler } from './handlers';
  import TagChip from './TagChip.svelte';

  interface Props {
    items: Transaction[];
    /** Card keywords from the Cards tab, to colour card tags. */
    cardKeywords: string[];
    selectedIds: string[];
    pageSelected: boolean;
    sortKey: SortKey;
    sortDir: SortDir;
    onsort: SortHandler;
    ontoggle: RowToggleHandler;
    ontogglepage: CheckedHandler;
    onedit: TransactionHandler;
    ondelete: TransactionHandler;
  }

  let {
    items,
    cardKeywords,
    selectedIds,
    pageSelected,
    sortKey,
    sortDir,
    onsort,
    ontoggle,
    ontogglepage,
    onedit,
    ondelete,
  }: Props = $props();

  const columns: { key: SortKey; label: string; align?: string }[] = [
    { key: 'date', label: 'Date' },
    { key: 'accountName', label: 'Account' },
    { key: 'payee', label: 'Payee' },
    { key: 'category', label: 'Category' },
    { key: 'payment', label: 'Payment', align: 'text-right' },
    { key: 'deposit', label: 'Deposit', align: 'text-right' },
  ];
</script>

<div class="-mx-5 overflow-x-auto">
  <table class="w-full min-w-[56rem] text-sm">
    <thead>
      <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
        <th class="py-2 pl-5">
          <input
            type="checkbox"
            class="size-4 accent-indigo-600"
            aria-label="Select all on this page"
            checked={pageSelected}
            onchange={(event) => ontogglepage(event.currentTarget.checked)}
          />
        </th>
        {#each columns as column (column.key)}
          <th
            class="px-5 py-2 font-medium {column.align ?? ''}"
            aria-sort={sortKey === column.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
          >
            <button
              type="button"
              class="-mx-1 inline-flex items-center gap-1 rounded-md px-1 uppercase hover:bg-slate-900/[0.06] hover:text-slate-900 active:bg-slate-900/10 dark:hover:bg-white/10 dark:hover:text-white"
              title="Sort by {column.label.toLowerCase()}"
              onclick={() => onsort(column.key)}
            >
              {column.label}
              {#if sortKey === column.key}<span aria-hidden="true">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
        {/each}
        <th class="px-5 py-2"><span class="sr-only">Actions</span></th>
      </tr>
    </thead>
    <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
      {#each items as tx (tx.id)}
        <tr class="align-top {selectedIds.includes(tx.id) ? 'bg-indigo-500/[0.06] dark:bg-indigo-400/10' : ''}">
          <td class="py-2.5 pl-5">
            <input
              type="checkbox"
              class="size-4 accent-indigo-600"
              aria-label="Select {tx.payee}"
              checked={selectedIds.includes(tx.id)}
              onchange={(event) => ontoggle(tx.id, event.currentTarget.checked)}
            />
          </td>
          <td class="tabular px-5 py-2.5 whitespace-nowrap">{formatDate(tx.date)}</td>
          <td class="px-5 py-2.5">{tx.accountName}</td>
          <td class="px-5 py-2.5">
            <p class="font-medium">{tx.payee}</p>
            {#if tx.memo}<p class="muted text-xs">{tx.memo}</p>{/if}
            {#if tx.tags.length}
              <div class="mt-1 flex flex-wrap gap-1">
                {#each tx.tags as tag (tag)}<TagChip {tag} {cardKeywords} />{/each}
              </div>
            {/if}
            {#if tx.issues?.length}
              <p class="mt-1 text-xs text-amber-600 dark:text-amber-400" title={tx.issues.join('\n')}>
                ⚠ {tx.issues.join('; ')}
              </p>
            {/if}
          </td>
          <td class="px-5 py-2.5"><CategoryBadge name={tx.category} type={tx.categoryType} /></td>
          <td class="px-5 py-2.5 text-right">
            {#if tx.payment}<Money value={-tx.payment} tone />{/if}
          </td>
          <td class="px-5 py-2.5 text-right">
            {#if tx.deposit}<Money value={tx.deposit} signed tone />{/if}
          </td>
          <td class="px-5 py-2 text-right whitespace-nowrap">
            <Button size="icon" icon="pencil" label="Edit {tx.payee}" onclick={() => onedit(tx)} />
            <Button size="icon" icon="trash" label="Delete {tx.payee}" onclick={() => ondelete(tx)} />
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
