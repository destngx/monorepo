<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { Account, CaptureResult, Category, TransactionBatchResult, TransactionInput } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import type { InboxItem } from './materials';
  import { commitBatch, learnedRules, reconcile, rowProblems, ruleKey, type ReviewRow } from './review';
  import ReconciliationCard from './ReconciliationCard.svelte';
  import RulesCard from './RulesCard.svelte';
  import SuggestionRow from './SuggestionRow.svelte';

  // Review and commit (WM-CAP-7/8/9/11). Nothing is written until Commit.
  interface Props {
    rows: ReviewRow[];
    result: CaptureResult;
    items: InboxItem[];
    accounts: Account[];
    categories: Category[];
    tags: string[];
    /** VND per USD for previewing USD and USDT rows; the sheet does the real conversion. */
    usdRate?: number | null;
    /** Writes the rows; the batch create path by default. Quick capture approves drafts instead. */
    write?: (batchId: string, rows: ReviewRow[], inputs: TransactionInput[]) => Promise<TransactionBatchResult>;
    /** Discards the selected rows; offered only when set (quick capture drafts). */
    reject?: (rows: ReviewRow[]) => Promise<void>;
    /** Button verb: "Commit" or "Approve". */
    verb?: string;
    emptyText?: string;
  }

  let {
    rows = $bindable(),
    result,
    items,
    accounts,
    categories,
    tags,
    usdRate = null,
    write = (batchId, _rows, inputs) => api.createTransactions(batchId, inputs),
    reject,
    verb = 'Commit',
    emptyText = 'No transactions were found in these materials.',
  }: Props = $props();

  const itemById = $derived(new Map(items.map((item) => [item.id, item])));
  const accountByName = $derived(new Map(accounts.map((account) => [account.name, account])));
  const activeAccounts = $derived(accounts.filter((account) => account.type !== 'deprecated'));
  const accountNames = $derived(accounts.map((account) => account.name));
  const categoryNames = $derived(categories.map((category) => category.name));
  const grouped = $derived(
    (['expense', 'income', 'non-budget'] as const).map((type) => ({
      type,
      items: categories.filter((category) => category.type === type),
    })),
  );

  const reconciliation = $derived(reconcile(result.balances, rows));
  const problems = $derived(
    rows.map((row) => (row.selected && !row.createdId ? rowProblems(row.tx, accountNames, categoryNames) : [])),
  );
  const selectable = $derived(rows.filter((row) => !row.createdId));
  const selectedCount = $derived(selectable.filter((row) => row.selected).length);
  const committedCount = $derived(rows.filter((row) => row.createdId).length);

  // A commit whose response never arrived is retried with the same batch id,
  // so the engine replays instead of writing twice (WM-TX-10).
  let pending = $state<{ batchId: string; indices: number[]; inputs: TransactionInput[] } | null>(null);
  let committing = $state(false);
  let rejecting = $state(false);
  const busy = $derived(committing || rejecting);
  const pendingIndices = $derived(new Set(pending?.indices ?? []));
  const locked = (index: number) => Boolean(rows[index].createdId) || pendingIndices.has(index) || busy;

  function selectAll(selected: boolean) {
    for (const row of rows) if (!row.createdId) row.selected = selected;
  }

  async function commit() {
    let batch = pending;
    if (!batch) {
      if (problems.some((list) => list.length)) {
        toasts.push('warning', `Fix the highlighted rows before you ${verb.toLowerCase()}.`);
        return;
      }
      batch = { batchId: crypto.randomUUID(), ...commitBatch(rows) };
      if (!batch.inputs.length) return;
    }
    pending = batch;
    committing = true;
    try {
      const outcome = await write(
        batch.batchId,
        batch.indices.map((index) => rows[index]),
        batch.inputs,
      );
      pending = null;
      for (const line of outcome.rows) {
        const row = rows[batch.indices[line.index]];
        if (line.id) {
          row.createdId = line.id;
          row.error = undefined;
        } else {
          row.error = line.error ?? 'Not written.';
        }
      }
      const failed = outcome.rows.filter((line) => !line.id).length;
      const created = outcome.rows.length - failed;
      if (failed)
        toasts.push(
          'warning',
          `${created} written, ${failed} failed. Fix the rows marked in red and ${verb.toLowerCase()} again.`,
        );
      else toasts.push('success', `${created} transaction${created === 1 ? '' : 's'} written to the sheet.`);
      await invalidateAll();
    } catch (error) {
      // A 4xx means the engine rejected the whole batch and wrote nothing.
      if (error instanceof ApiError && error.status >= 400 && error.status < 500) pending = null;
      toasts.error(error, pending ? 'No response from the engine; retry is safe' : `${verb} failed`);
    } finally {
      committing = false;
    }
  }

  async function rejectSelected() {
    if (!reject) return;
    const chosen = rows.filter((row, index) => row.selected && !row.createdId && !pendingIndices.has(index));
    if (!chosen.length) return;
    rejecting = true;
    try {
      await reject(chosen);
      rows = rows.filter((row) => !chosen.includes(row));
      toasts.push('success', `${chosen.length} draft${chosen.length === 1 ? '' : 's'} rejected.`);
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Reject failed');
    } finally {
      rejecting = false;
    }
  }

  // Learned rules (WM-CAP-11): offered after rows are written.
  let savedRules = $state<string[]>([]);
  let skippedRules = $state<string[]>([]);
  let savingRules = $state(false);
  const rules = $derived(learnedRules(rows).filter((rule) => !savedRules.includes(ruleKey(rule))));
  const rulesToSave = $derived(rules.filter((rule) => !skippedRules.includes(ruleKey(rule))));

  async function saveRules() {
    savingRules = true;
    try {
      const saved = await api.saveCaptureRules(rulesToSave);
      savedRules = [...savedRules, ...rulesToSave.map(ruleKey)];
      toasts.push('success', `Rules saved: ${saved.created} new, ${saved.updated} updated.`);
    } catch (error) {
      toasts.error(error, 'Could not save the rules');
    } finally {
      savingRules = false;
    }
  }

  /** VND equivalent of a USD or USDT row, rendered through Money so it masks (WM-UI-2). */
  function usdPreview(row: ReviewRow): number | null {
    const currency = accountByName.get(row.tx.accountName)?.currency;
    if (!usdRate || (currency !== 'USD' && currency !== 'USDT')) return null;
    const amount = row.tx.payment ?? row.tx.deposit ?? 0;
    return amount * usdRate;
  }
</script>

{#if result.unreadable.length}
  <Card title="Not read" subtitle="These materials produced no transactions. Nothing was guessed." class="mb-4">
    <ul class="space-y-1 text-sm">
      {#each result.unreadable as entry (entry.materialId)}
        <li>
          <strong>{entry.materialId}</strong>
          {itemById.get(entry.materialId)?.name ?? ''}: <span class="muted">{entry.reason}</span>
        </li>
      {/each}
    </ul>
  </Card>
{/if}

{#if reconciliation.length}
  <ReconciliationCard {reconciliation} />
{/if}

<Card
  title="Suggested transactions"
  subtitle="{rows.length} found · {selectedCount} selected{committedCount
    ? ` · ${committedCount} written`
    : ''}. Highlighted fields need a closer look."
>
  {#snippet actions()}
    <button type="button" class="btn-ghost" disabled={busy} onclick={() => selectAll(true)}>Select all</button>
    <button type="button" class="btn-ghost" disabled={busy} onclick={() => selectAll(false)}>None</button>
  {/snippet}

  {#if rows.length}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[68rem] text-sm">
        <thead>
          <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
            <th class="w-8 px-3 py-2"><span class="sr-only">Selected</span></th>
            <th class="px-2 py-2 font-medium">Date</th>
            <th class="px-2 py-2 font-medium">Account</th>
            <th class="px-2 py-2 font-medium">Payee</th>
            <th class="px-2 py-2 font-medium">Category</th>
            <th class="px-2 py-2 text-right font-medium">Payment</th>
            <th class="px-2 py-2 text-right font-medium">Deposit</th>
            <th class="px-3 py-2 font-medium">Confidence</th>
          </tr>
        </thead>
        {#each rows as row, index (row.suggestion.id)}
          <SuggestionRow
            bind:row={rows[index]}
            disabled={locked(index)}
            problems={problems[index]}
            usdPreview={usdPreview(row)}
            {itemById}
            {activeAccounts}
            {categoryNames}
            {grouped}
            {tags}
          />
        {/each}
      </table>
    </div>
  {:else}
    <p class="muted text-sm">{emptyText}</p>
  {/if}

  <div class="mt-4 flex flex-wrap items-center justify-end gap-3">
    {#if pending && !committing}
      <p class="muted text-xs">
        The last {verb.toLowerCase()} got no response. Retrying sends the same rows and cannot write them twice.
      </p>
    {/if}
    {#if reject}
      <button
        type="button"
        class="btn-ghost"
        disabled={busy || selectedCount === 0 || Boolean(pending)}
        onclick={rejectSelected}
      >
        {rejecting ? 'Rejecting…' : `Reject ${selectedCount}`}
      </button>
    {/if}
    <button type="button" class="btn-primary" disabled={busy || (!pending && selectedCount === 0)} onclick={commit}>
      {committing
        ? 'Writing…'
        : pending
          ? `Retry ${verb.toLowerCase()}`
          : `${verb} ${selectedCount} row${selectedCount === 1 ? '' : 's'}`}
    </button>
  </div>
</Card>

{#if rules.length}
  <RulesCard
    {rules}
    bind:skipped={skippedRules}
    saving={savingRules}
    saveCount={rulesToSave.length}
    onsave={saveRules}
  />
{/if}
