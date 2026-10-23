<script lang="ts">
  import { openOnPendingFill } from '$lib/fill/pending.svelte';
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { untrack } from 'svelte';
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { api } from '$lib/api/endpoints';
  import {
    track,
    Button,
    Card,
    Deferred,
    EmptyState,
    Icon,
    MenuButton,
    Modal,
    SkeletonTable,
    SkeletonText,
    type MenuItem,
  } from '@libs/ui-kit';
  import type { Transaction, TransactionInput } from '$lib/api/types';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatDate } from '$lib/format';
  import { lastUsedAccount } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import BulkBar from '$lib/transactions/BulkBar.svelte';
  import BulkFailures from '$lib/transactions/BulkFailures.svelte';
  import { billedCards } from '$lib/transactions/cards';
  import TransactionForm from '$lib/transactions/TransactionForm.svelte';
  import TransactionTable from '$lib/transactions/TransactionTable.svelte';
  import { nextEntry, prefillFromURL, recategorise, type BulkFailure } from '$lib/transactions/entry';
  import {
    needsCategory,
    paginate,
    searchTransactions,
    sortTransactions,
    type SortDir,
    type SortKey,
  } from '$lib/transactions/filter';

  // The header, search and Add render at once; the ledger streams into the
  // table and the form options (accounts, categories, tags) into the form.
  let { data } = $props();
  const ledger = track(() => data.transactions);
  const options = track(() => data.options);
  const cardConfig = track(() => data.cardConfig);
  const rows = $derived(ledger.value ?? []);
  const accounts = $derived(options.value?.accounts ?? []);
  const categories = $derived(options.value?.categories ?? []);
  const cards = $derived(cardConfig.value ? billedCards(cardConfig.value, accounts) : []);

  const PAGE_SIZE = 50;
  let query = $state('');
  let sortKey = $state<SortKey>('date');
  let sortDir = $state<SortDir>('desc');
  let pageNumber = $state(1);

  // `?uncategorized=1` (from the home inbox) lists the rows that still need a category.
  const onlyUncategorized = $derived(page.url.searchParams.get('uncategorized') === '1');
  const scoped = $derived(onlyUncategorized ? rows.filter(needsCategory) : rows);
  const filtered = $derived(sortTransactions(searchTransactions(scoped, query), sortKey, sortDir));

  function showAllRows() {
    const url = new URL(page.url);
    url.searchParams.delete('uncategorized');
    pageNumber = 1;
    void goto(url, { replaceState: true, keepFocus: true, noScroll: true });
  }
  const paged = $derived(paginate(filtered, pageNumber, PAGE_SIZE));

  // Editor: null = closed, 'new' = add, otherwise the row being edited.
  let editing = $state<Transaction | 'new' | null>(null);
  let deleting = $state<Transaction | null>(null);
  let busy = $state(false);
  let removing = $state(false);
  // Seed for a new row, and a counter that remounts the form for "Save & add another".
  let prefill = $state<Partial<TransactionInput>>({});
  let formKey = $state(0);

  function openNew(seed: Partial<TransactionInput> = {}) {
    prefill = seed;
    formKey++;
    editing = 'new';
  }

  // A chat fill opens its form here (WM-FILL-7).
  openOnPendingFill('transaction', () => openNew());

  // One Add menu: type a row in, or read it from a screenshot or statement.
  const addItems: MenuItem[] = [
    { label: 'Add transaction', icon: 'plus', onselect: () => openNew() },
    { label: 'Smart capture', icon: 'camera', onselect: () => void goto(`${base}/transactions/capture`) },
  ];

  const editorKey = $derived(editing === null ? '' : editing === 'new' ? `new-${formKey}` : editing.id);

  function closeEditor() {
    editing = null;
    prefill = {};
  }

  // Deep link `/transactions?new=1&account=&payee=&category=` opens a prefilled
  // form (WM-TX-9), e.g. a balance adjustment from the accounts page; the
  // parameters are then dropped so a reload does not reopen it.
  // It waits for the form options, so unknown accounts and categories are dropped.
  $effect(() => {
    if (!options.value) return;
    const seed = prefillFromURL(
      page.url.searchParams,
      options.value.accounts.map((account) => account.name),
      options.value.categories.map((category) => category.name),
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
  const selectedRows = $derived(rows.filter((tx) => selectedIds.includes(tx.id)));
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

  // Runs from the bulk bar's Button (busy until done; an unexpected error is toasted there).
  async function applyCategory() {
    if (!bulkCategory || !selectedRows.length) return;
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
      // Fresh etags for the next attempt, whatever happened.
      await invalidateAll();
    }
  }

  const defaultAccount = $derived(accounts.some((a) => a.name === lastUsedAccount.value) ? lastUsedAccount.value : '');

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

  // Runs from the dialog's Delete Button, which stays busy until the sheet answers.
  async function remove() {
    if (!deleting || removing) return;
    removing = true;
    try {
      await api.deleteTransaction(deleting.id, deleting.etag);
      toasts.push('success', 'Transaction deleted.');
      deleting = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not delete the transaction');
      await invalidateAll();
    } finally {
      removing = false;
    }
  }
</script>

<PageHeader
  title="Transactions"
  subtitle={ledger.value ? `${ledger.value.length.toLocaleString('vi-VN')} transactions` : 'Counting transactions…'}
>
  {#snippet actions()}
    <!-- WM-TX-6 -->
    <a class="btn-ghost" href="{base}/transactions/drafts"
      ><Icon name="inbox" size={16} /> Drafts{pendingDrafts.count ? ` (${pendingDrafts.count})` : ''}</a
    >
    <MenuButton variant="primary" label="Add" icon="plus" items={addItems} />
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
    {#if onlyUncategorized}
      <Button class="text-sm" title="Show all transactions" onclick={showAllRows}
        >Without a category{ledger.value ? ` (${scoped.length})` : ''}
        <Icon name="x" size={14} /><span class="sr-only">Show all</span></Button
      >
    {/if}
    {#if query && ledger.value}<p class="muted text-sm" aria-live="polite">{filtered.length} matches</p>{/if}
  </div>

  <Deferred data={data.transactions} label="transactions">
    {#snippet children()}
      <div>
        {#if selectedIds.length}
          <BulkBar
            count={selectedIds.length}
            {categories}
            loading={options.loading}
            bind:category={bulkCategory}
            onapply={applyCategory}
            onclear={() => (selectedIds = [])}
          />
        {/if}

        {#if bulkFailures.length}
          <BulkFailures failures={bulkFailures} ondismiss={() => (bulkFailures = [])} />
        {/if}

        {#if paged.items.length}
          <TransactionTable
            items={paged.items}
            {cards}
            {selectedIds}
            {pageSelected}
            {sortKey}
            {sortDir}
            onsort={sortBy}
            ontoggle={toggleRow}
            ontogglepage={togglePage}
            onedit={(tx) => (editing = tx)}
            ondelete={(tx) => (deleting = tx)}
          />

          {#if paged.pages > 1}
            <nav class="mt-4 flex items-center justify-between text-sm" aria-label="Pagination">
              <Button icon="chevronLeft" disabled={paged.page === 1} onclick={() => (pageNumber = paged.page - 1)}
                >Previous</Button
              >
              <span class="muted">Page {paged.page} of {paged.pages}</span>
              <Button disabled={paged.page === paged.pages} onclick={() => (pageNumber = paged.page + 1)}
                >Next <Icon name="chevronRight" size={16} /></Button
              >
            </nav>
          {/if}
        {:else}
          <EmptyState
            title={query
              ? 'No matching transactions'
              : onlyUncategorized
                ? 'Every transaction has a category'
                : 'No transactions yet'}
            detail={query
              ? 'Try another search.'
              : onlyUncategorized
                ? 'Nothing left to categorize.'
                : 'Add the first one with the Add button.'}
          />
        {/if}
      </div>
    {/snippet}
    {#snippet skeleton()}<SkeletonTable rows={8} columns={7} label="transactions" />{/snippet}
  </Deferred>
</Card>

<Button
  size="icon"
  icon="arrowUp"
  label="Scroll to top"
  class="panel fixed right-4 bottom-40 z-20 size-10 lg:bottom-20"
  onclick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}
/>

<!-- WM-TX-7 -->
<RecommendationsPanel
  title="Recommendations"
  subtitle="From the weekly review of the last 7 days"
  page="/transactions"
  class="mt-4"
  review={(force, signal) => api.review('transaction-review', {}, { force, signal })}
/>

<Modal
  open={editing !== null}
  title={editing === 'new' ? 'Add transaction' : 'Edit transaction'}
  onclose={closeEditor}
  {busy}
  wide
>
  <!-- The form needs the accounts, categories and tags; it opens at once and they stream in. -->
  <Deferred data={data.options} label="accounts, categories and tags">
    {#snippet children(value)}
      {#key editorKey}
        <TransactionForm
          accounts={value.accounts}
          categories={value.categories}
          tags={value.tags}
          {cards}
          initial={editing === 'new' ? null : editing}
          {defaultAccount}
          {prefill}
          {busy}
          onsubmit={save}
          oncancel={closeEditor}
        />
      {/key}
    {/snippet}
    {#snippet skeleton()}<SkeletonText lines={6} label="accounts, categories and tags" />{/snippet}
  </Deferred>
</Modal>

<Modal open={deleting !== null} title="Delete transaction" onclose={() => (deleting = null)} busy={removing}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.payee}</strong> on {formatDate(deleting.date)} from {deleting.accountName}? The row is
      cleared in the sheet.
    </p>
  {/if}
  {#snippet footer()}
    <Button disabled={removing} onclick={() => (deleting = null)}>Cancel</Button>
    <Button variant="danger" busy={removing} busyText="Deleting…" onclick={remove}>Delete</Button>
  {/snippet}
</Modal>
