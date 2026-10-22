<script lang="ts">
  import type { Transaction } from '$lib/api/types';
  import { Button } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import CategoryBadge from './CategoryBadge.svelte';
  import type { SortDir, SortKey } from './filter';
  import type { CheckedHandler, RowToggleHandler, SortHandler, TransactionHandler } from './handlers';
  import Badge from '$lib/components/Badge.svelte';
  import { mccTag, paidWith, type BilledCard } from './cards';

  interface Props {
    items: Transaction[];
    /** Held cards, to name the card that paid. */
    cards: BilledCard[];
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
    cards,
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

<!-- On phones each row stacks into a card (payee and amount, account and category, date and
     actions) and the header becomes a row of sort buttons; from sm up it is the full table. -->
<div class="-mx-5 overflow-x-auto">
  <table class="block w-full text-sm sm:table sm:min-w-[56rem]">
    <thead class="block sm:table-header-group">
      <tr
        class="muted flex flex-wrap items-center gap-x-4 gap-y-1 border-b border-stone-900/10 px-5 pb-2 text-left text-xs uppercase sm:table-row sm:p-0 dark:border-white/10"
      >
        <th class="block sm:table-cell sm:py-2 sm:pl-5">
          <input
            type="checkbox"
            class="size-4 accent-accent-600"
            aria-label="Select all on this page"
            checked={pageSelected}
            onchange={(event) => ontogglepage(event.currentTarget.checked)}
          />
        </th>
        {#each columns as column (column.key)}
          <th
            class="block font-medium sm:table-cell sm:px-5 sm:py-2 {column.align ?? ''}"
            aria-sort={sortKey === column.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
          >
            <button
              type="button"
              class="-mx-1 inline-flex items-center gap-1 rounded-md px-1 uppercase hover:bg-stone-900/[0.06] hover:text-stone-900 active:bg-stone-900/10 dark:hover:bg-white/10 dark:hover:text-white"
              title="Sort by {column.label.toLowerCase()}"
              onclick={() => onsort(column.key)}
            >
              {column.label}
              {#if sortKey === column.key}<span aria-hidden="true">{sortDir === 'asc' ? '▲' : '▼'}</span>{/if}
            </button>
          </th>
        {/each}
        <th class="hidden sm:table-cell sm:px-5 sm:py-2"><span class="sr-only">Actions</span></th>
      </tr>
    </thead>
    <tbody class="block divide-y divide-stone-900/5 sm:table-row-group dark:divide-white/5">
      {#each items as tx (tx.id)}
        {@const card = paidWith(tx, cards)}
        {@const mcc = mccTag(tx.tags)}
        <tr
          class="grid grid-cols-[auto_minmax(0,1fr)_auto] gap-x-3 gap-y-1 px-5 py-3 align-top sm:table-row sm:p-0 {selectedIds.includes(
            tx.id,
          )
            ? 'bg-accent-500/[0.06] dark:bg-accent-400/10'
            : ''}"
        >
          <td class="row-span-3 pt-0.5 sm:table-cell sm:py-2.5 sm:pl-5">
            <input
              type="checkbox"
              class="size-4 accent-accent-600"
              aria-label="Select {tx.payee}"
              checked={selectedIds.includes(tx.id)}
              onchange={(event) => ontoggle(tx.id, event.currentTarget.checked)}
            />
          </td>
          <td
            class="tabular muted col-start-2 row-start-3 text-xs whitespace-nowrap sm:table-cell sm:px-5 sm:py-2.5 sm:text-sm sm:text-inherit"
          >
            {formatDate(tx.date)}
          </td>
          <td class="col-start-2 row-start-2 text-xs sm:table-cell sm:px-5 sm:py-2.5 sm:text-sm">
            {tx.accountName}
            {#if card || mcc}
              <p class="muted text-xs">{[card?.name, mcc && `MCC ${mcc}`].filter(Boolean).join(' · ')}</p>
            {/if}
          </td>
          <td class="col-start-2 row-start-1 min-w-0 break-words sm:table-cell sm:px-5 sm:py-2.5">
            <p class="font-medium">{tx.payee}</p>
            {#if tx.memo}<p class="muted text-xs">{tx.memo}</p>{/if}
            {#if tx.tags.length}
              <div class="mt-1 flex flex-wrap gap-1">
                {#each tx.tags as tag (tag)}<Badge>{tag}</Badge>{/each}
              </div>
            {/if}
            {#if tx.issues?.length}
              <p class="mt-1 text-xs text-amber-600 dark:text-amber-400" title={tx.issues.join('\n')}>
                ⚠ {tx.issues.join('; ')}
              </p>
            {/if}
          </td>
          <td class="col-start-3 row-start-2 justify-self-end text-right sm:table-cell sm:px-5 sm:py-2.5 sm:text-left">
            <CategoryBadge name={tx.category} type={tx.categoryType} />
          </td>
          <td class="col-start-3 row-start-1 justify-self-end text-right sm:table-cell sm:px-5 sm:py-2.5">
            {#if tx.payment}<Money value={-tx.payment} tone />{/if}
          </td>
          <td class="col-start-3 row-start-1 justify-self-end text-right sm:table-cell sm:px-5 sm:py-2.5">
            {#if tx.deposit}<Money value={tx.deposit} signed tone />{/if}
          </td>
          <td
            class="col-start-3 row-start-3 -my-1 justify-self-end text-right whitespace-nowrap sm:my-0 sm:table-cell sm:px-5 sm:py-2"
          >
            <Button size="icon" icon="pencil" label="Edit {tx.payee}" onclick={() => onedit(tx)} />
            <Button size="icon" icon="trash" label="Delete {tx.payee}" onclick={() => ondelete(tx)} />
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
