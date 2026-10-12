<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { untrack } from 'svelte';
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { api } from '$lib/api/endpoints';
  import type { Transaction, TransactionInput } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatDate } from '$lib/format';
  import { lastUsedAccount } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import CategoryBadge from '$lib/transactions/CategoryBadge.svelte';
  import TagChip from '$lib/transactions/TagChip.svelte';
  import TransactionForm from '$lib/transactions/TransactionForm.svelte';
  import { nextEntry, prefillFromURL, recategorise, type BulkFailure } from '$lib/transactions/entry';
  import { paginate, searchTransactions, sortTransactions, type SortDir, type SortKey } from '$lib/transactions/filter';

  let { data } = $props();

  const PAGE_SIZE = 50;
  let query = $state('');
  let sortKey = $state<SortKey>('date');
  let sortDir = $state<SortDir>('desc');
  let pageNumber = $state(1);

  const filtered = $derived(sortTransactions(searchTransactions(data.transactions, query), sortKey, sortDir));
  const paged = $derived(paginate(filtered, pageNumber, PAGE_SIZE));

  // Editor: null = closed, 'new' = add, otherwise the row being edited.
  let editing = $state<Transaction | 'new' | null>(null);
  let deleting = $state<Transaction | null>(null);
  let busy = $state(false);
  // Seed for a new row, and a counter that remounts the form for "Save & add another".
  let prefill = $state<Partial<TransactionInput>>({});
  let formKey = $state(0);

  function openNew(seed: Partial<TransactionInput> = {}) {
    prefill = seed;
    formKey++;
    editing = 'new';
  }

  const editorKey = $derived(editing === null ? '' : editing === 'new' ? `new-${formKey}` : editing.id);

  function closeEditor() {
    editing = null;
    prefill = {};
  }

  // Deep link `/transactions?new=1&account=&payee=&category=` opens a prefilled
  // form (WM-TX-9), e.g. a balance adjustment from the accounts page; the
  // parameters are then dropped so a reload does not reopen it.
  $effect(() => {
    const seed = prefillFromURL(
      page.url.searchParams,
      data.accounts.map((account) => account.name),
      data.categories.map((category) => category.name),
    );
    if (!seed) return;
    untrack(() => openNew(seed));
    const url = new URL(page.url);
    for (const key of ['new', 'account', 'payee', 'category']) url.searchParams.delete(key);
    void goto(url, { replaceState: true, keepFocus: true, noScroll: true });
  });

  // Bulk recategorise (WM-TX-9).
  let selectedIds = $state<string[]>([]);
  let bulkCategory = $state('');
  let bulkFailures = $state<BulkFailure[]>([]);
  const selectedRows = $derived(data.transactions.filter((tx) => selectedIds.includes(tx.id)));
  const pageIds = $derived(paged.items.map((tx) => tx.id));
  const pageSelected = $derived(pageIds.length > 0 && pageIds.every((id) => selectedIds.includes(id)));

  function toggleRow(id: string, checked: boolean) {
    selectedIds = checked ? [...selectedIds, id] : selectedIds.filter((selected) => selected !== id);
  }

  function togglePage(checked: boolean) {
    selectedIds = checked
      ? [...new Set([...selectedIds, ...pageIds])]
      : selectedIds.filter((id) => !pageIds.includes(id));
  }

  async function applyCategory() {
    if (!bulkCategory || !selectedRows.length) return;
    busy = true;
    bulkFailures = [];
    try {
      const result = await recategorise(selectedRows, bulkCategory, api.updateTransaction);
      bulkFailures = result.failed;
      selectedIds = result.failed.map((failure) => failure.id);
      if (result.updated.length) {
        toasts.push('success', `${result.updated.length} transaction(s) moved to ${bulkCategory}.`);
      }
      if (result.failed.length) toasts.push('warning', `${result.failed.length} transaction(s) were not changed.`);
    } finally {
      busy = false;
      // Fresh etags for the next attempt, whatever happened.
      await invalidateAll();
    }
  }

  const defaultAccount = $derived(
    data.accounts.some((a) => a.name === lastUsedAccount.value) ? lastUsedAccount.value : '',
  );

  function sortBy(key: SortKey) {
    if (sortKey === key) sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    else {
      sortKey = key;
      sortDir = key === 'date' || key === 'payment' || key === 'deposit' ? 'desc' : 'asc';
    }
    pageNumber = 1;
  }

  async function save(input: TransactionInput, addAnother: boolean) {
    busy = true;
    try {
      if (editing && editing !== 'new') {
        await api.updateTransaction(editing.id, editing.etag, input);
        toasts.push('success', 'Transaction updated.');
      } else {
        await api.createTransaction(input);
        toasts.push('success', 'Transaction added.');
      }
      lastUsedAccount.value = input.accountName;
      if (addAnother) openNew(nextEntry(input));
      else closeEditor();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not save the transaction');
      // A conflict means the row changed in the sheet; reload so the next edit is fresh.
      await invalidateAll();
    } finally {
      busy = false;
    }
  }

  async function remove() {
    if (!deleting) return;
    busy = true;
    try {
      await api.deleteTransaction(deleting.id, deleting.etag);
      toasts.push('success', 'Transaction deleted.');
      deleting = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not delete the transaction');
      await invalidateAll();
    } finally {
      busy = false;
    }
  }

  const columns: { key: SortKey; label: string; align?: string }[] = [
    { key: 'date', label: 'Date' },
    { key: 'accountName', label: 'Account' },
    { key: 'payee', label: 'Payee' },
    { key: 'category', label: 'Category' },
    { key: 'payment', label: 'Payment', align: 'text-right' },
    { key: 'deposit', label: 'Deposit', align: 'text-right' },
  ];
</script>

<PageHeader title="Transactions" subtitle="{data.transactions.length.toLocaleString('vi-VN')} transactions">
  {#snippet actions()}
    <!-- WM-TX-6 -->
    <a class="btn-ghost" href="{base}/transactions/capture"><Icon name="camera" size={16} /> Add from materials</a>
    <button type="button" class="btn-primary" onclick={() => openNew()}><Icon name="plus" size={16} /> Add</button>
  {/snippet}
</PageHeader>

<Card>
  <div class="mb-4 flex flex-wrap items-center gap-3">
    <label class="relative min-w-60 flex-1">
      <span class="sr-only">Search transactions</span>
      <Icon name="search" size={16} class="muted absolute top-1/2 left-3 -translate-y-1/2" />
      <input
        type="search"
        class="field pl-9"
        placeholder="Search payee, account, category, memo, tags"
        bind:value={query}
        oninput={() => (pageNumber = 1)}
      />
    </label>
    {#if query}<p class="muted text-sm">{filtered.length} matches</p>{/if}
  </div>

  {#if selectedIds.length}
    <div
      class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-indigo-500/30 bg-indigo-500/5 px-3 py-2 text-sm"
    >
      <span class="font-medium">{selectedIds.length} selected</span>
      <label class="sr-only" for="bulk-category">New category</label>
      <select id="bulk-category" class="field w-auto min-w-48" bind:value={bulkCategory}>
        <option value="" disabled>Move to category</option>
        {#each data.categories as category (category.name)}
          <option value={category.name}>{category.name}</option>
        {/each}
      </select>
      <button type="button" class="btn-primary" disabled={busy || !bulkCategory} onclick={applyCategory}
        >{busy ? 'Updating…' : 'Recategorise'}</button
      >
      <button type="button" class="btn-ghost" disabled={busy} onclick={() => (selectedIds = [])}>Clear</button>
    </div>
  {/if}

  {#if bulkFailures.length}
    <div role="alert" class="mb-4 rounded-xl border border-orange-500/40 p-3 text-sm">
      <div class="flex items-start justify-between gap-2">
        <p class="font-medium text-orange-700 dark:text-orange-300">These were not recategorised and stay selected:</p>
        <button
          type="button"
          class="opacity-60 hover:opacity-100"
          aria-label="Dismiss"
          onclick={() => (bulkFailures = [])}><Icon name="x" size={16} /></button
        >
      </div>
      <ul class="mt-1 space-y-0.5">
        {#each bulkFailures as failure (failure.id)}
          <li><span class="font-medium">{failure.payee}</span> ({failure.id}): {failure.message}</li>
        {/each}
      </ul>
    </div>
  {/if}

  {#if paged.items.length}
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
                onchange={(event) => togglePage(event.currentTarget.checked)}
              />
            </th>
            {#each columns as column (column.key)}
              <th
                class="px-5 py-2 font-medium {column.align ?? ''}"
                aria-sort={sortKey === column.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : 'none'}
              >
                <button
                  type="button"
                  class="inline-flex items-center gap-1 uppercase"
                  onclick={() => sortBy(column.key)}
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
          {#each paged.items as tx (tx.id)}
            <tr class="align-top hover:bg-slate-900/[0.02] dark:hover:bg-white/[0.02]">
              <td class="py-2.5 pl-5">
                <input
                  type="checkbox"
                  class="size-4 accent-indigo-600"
                  aria-label="Select {tx.payee}"
                  checked={selectedIds.includes(tx.id)}
                  onchange={(event) => toggleRow(tx.id, event.currentTarget.checked)}
                />
              </td>
              <td class="tabular px-5 py-2.5 whitespace-nowrap">{formatDate(tx.date)}</td>
              <td class="px-5 py-2.5">{tx.accountName}</td>
              <td class="px-5 py-2.5">
                <p class="font-medium">{tx.payee}</p>
                {#if tx.memo}<p class="muted text-xs">{tx.memo}</p>{/if}
                {#if tx.tags.length}
                  <div class="mt-1 flex flex-wrap gap-1">
                    {#each tx.tags as tag (tag)}<TagChip {tag} />{/each}
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
                <button type="button" class="btn-ghost px-2" aria-label="Edit {tx.payee}" onclick={() => (editing = tx)}
                  ><Icon name="pencil" size={15} /></button
                >
                <button
                  type="button"
                  class="btn-ghost px-2"
                  aria-label="Delete {tx.payee}"
                  onclick={() => (deleting = tx)}><Icon name="trash" size={15} /></button
                >
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    {#if paged.pages > 1}
      <nav class="mt-4 flex items-center justify-between text-sm" aria-label="Pagination">
        <button
          type="button"
          class="btn-ghost"
          disabled={paged.page === 1}
          onclick={() => (pageNumber = paged.page - 1)}
        >
          <Icon name="chevronLeft" size={16} /> Previous
        </button>
        <span class="muted">Page {paged.page} of {paged.pages}</span>
        <button
          type="button"
          class="btn-ghost"
          disabled={paged.page === paged.pages}
          onclick={() => (pageNumber = paged.page + 1)}
        >
          Next <Icon name="chevronRight" size={16} />
        </button>
      </nav>
    {/if}
  {:else}
    <EmptyState
      title={query ? 'No matching transactions' : 'No transactions yet'}
      detail={query ? 'Try another search.' : 'Add the first one with the Add button.'}
    />
  {/if}
</Card>

<button
  type="button"
  class="glass fixed right-4 bottom-4 z-20 grid size-10 place-items-center"
  aria-label="Scroll to top"
  onclick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}><Icon name="arrowUp" size={18} /></button
>

<!-- WM-TX-7 -->
<InsightCard
  title="AI weekly review"
  class="mt-4"
  load={(force, signal) => api.review('transaction-review', {}, { force, signal })}
/>

<Modal
  open={editing !== null}
  title={editing === 'new' ? 'Add transaction' : 'Edit transaction'}
  onclose={closeEditor}
  wide
>
  {#key editorKey}
    <TransactionForm
      accounts={data.accounts}
      categories={data.categories}
      tags={data.tags}
      initial={editing === 'new' ? null : editing}
      {defaultAccount}
      {prefill}
      {busy}
      onsubmit={save}
      oncancel={closeEditor}
    />
  {/key}
</Modal>

<Modal open={deleting !== null} title="Delete transaction" onclose={() => (deleting = null)}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.payee}</strong> on {formatDate(deleting.date)} from {deleting.accountName}? The row is
      cleared in the sheet.
    </p>
  {/if}
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (deleting = null)}>Cancel</button>
    <button type="button" class="btn-danger" disabled={busy} onclick={remove}>{busy ? 'Deleting…' : 'Delete'}</button>
  {/snippet}
</Modal>
