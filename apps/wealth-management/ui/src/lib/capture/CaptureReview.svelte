<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { Account, CaptureResult, Category, TransactionBatchResult, TransactionInput } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import type { InboxItem } from './materials';
  import {
    commitBatch,
    isRowDirty,
    learnedRules,
    projectBalances,
    requiredCardTags,
    reconcile,
    rowKey,
    rowProblems,
    ruleKey,
    runningBalances,
    type ReviewRow,
  } from './review';
  import BalancePanel from './BalancePanel.svelte';
  import RulesCard from './RulesCard.svelte';
  import SuggestionRow from './SuggestionRow.svelte';

  // Review and commit (WM-CAP-7/8/9/11). Nothing is written until Commit,
  // for the selection or one row at a time.
  interface Props {
    rows: ReviewRow[];
    result: CaptureResult;
    items: InboxItem[];
    accounts: Account[];
    categories: Category[];
    tags: string[];
    /** Open card keywords per account (cardTagOptions): spending on a card account must pick one. */
    cardTags: Map<string, string[]>;
    /** VND per USD for previewing USD and USDT rows; the sheet does the real conversion. */
    usdRate?: number | null;
    /** Writes the rows; the batch create path by default. Quick capture approves drafts instead. */
    write?: (batchId: string, rows: ReviewRow[], inputs: TransactionInput[]) => Promise<TransactionBatchResult>;
    /** Discards the selected rows; offered only when set (quick capture drafts). */
    reject?: (rows: ReviewRow[]) => Promise<void>;
    /** Saves one row's edits without writing it; offered only when set (quick capture drafts). */
    save?: (row: ReviewRow) => Promise<void>;
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
    cardTags,
    usdRate = null,
    write = (batchId, _rows, inputs) => api.createTransactions(batchId, inputs),
    reject,
    save,
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
  const projections = $derived(projectBalances(accounts, rows));
  const balancesAfter = $derived(runningBalances(accounts, rows));
  // Rows the user tried to write alone show their problems even when not selected.
  let attempted = $state<string[]>([]);
  const problems = $derived(
    rows.map((row) =>
      (row.selected || attempted.includes(row.suggestion.id)) && !row.createdId
        ? rowProblems(row.tx, accountNames, categoryNames, requiredCardTags(row, cardTags))
        : [],
    ),
  );
  const selectable = $derived(rows.filter((row) => !row.createdId));
  const selectedCount = $derived(selectable.filter((row) => row.selected).length);
  const committedCount = $derived(rows.filter((row) => row.createdId).length);

  // A commit whose response never arrived is retried with the same batch id,
  // so the engine replays instead of writing twice (WM-TX-10). It names its
  // rows by id, so a reload may reorder the list meanwhile.
  let pending = $state<{ batchId: string; ids: string[]; rows: ReviewRow[]; inputs: TransactionInput[] } | null>(null);
  let committing = $state(false);
  // A one-row write shows busy on that row's button, not on the bulk one.
  let committingRow = $state(false);
  let rejecting = $state(false);
  let saving = $state<string[]>([]);
  const busy = $derived(committing || rejecting);
  const pendingIds = $derived(new Set(pending?.ids ?? []));
  const locked = (row: ReviewRow) =>
    Boolean(row.createdId) || pendingIds.has(row.suggestion.id) || busy || saving.includes(row.suggestion.id);

  function selectAll(selected: boolean) {
    for (const row of rows) if (!row.createdId) row.selected = selected;
  }

  /** Writes the selection, or only the rows given; an unanswered batch is retried first. */
  async function commit(only?: number[]) {
    let batch = pending;
    if (!batch) {
      if (only) attempted = [...new Set([...attempted, ...only.map((index) => rows[index].suggestion.id)])];
      const checked = only ?? rows.flatMap((row, index) => (row.selected ? [index] : []));
      if (
        checked.some(
          (index) =>
            !rows[index].createdId &&
            rowProblems(rows[index].tx, accountNames, categoryNames, requiredCardTags(rows[index], cardTags)).length,
        )
      ) {
        toasts.push('warning', `Fix the highlighted rows before you ${verb.toLowerCase()}.`);
        return;
      }
      const { indices, inputs } = commitBatch(rows, only);
      if (!inputs.length) return;
      const chosen = indices.map((index) => $state.snapshot(rows[index]));
      batch = { batchId: crypto.randomUUID(), ids: chosen.map((row) => row.suggestion.id), rows: chosen, inputs };
    }
    pending = batch;
    committing = true;
    committingRow = Boolean(only);
    try {
      const outcome = await write(batch.batchId, batch.rows, batch.inputs);
      pending = null;
      for (const line of outcome.rows) {
        // A reload may have dropped the row meanwhile: then the sheet already shows it.
        const id = batch.ids[line.index];
        const row = rows.find((other) => other.suggestion.id === id);
        if (!row) continue;
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

  async function rejectRows(chosen: ReviewRow[]) {
    if (!reject || !chosen.length) return;
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

  const rejectSelected = () =>
    rejectRows(rows.filter((row) => row.selected && !row.createdId && !pendingIds.has(row.suggestion.id)));

  async function saveRow(row: ReviewRow) {
    if (!save) return;
    const id = row.suggestion.id;
    const key = rowKey(row);
    saving = [...saving, id];
    try {
      await save(row);
      row.savedKey = key;
      toasts.push('success', 'Draft saved.');
    } catch (error) {
      toasts.error(error, 'Could not save the draft');
    } finally {
      saving = saving.filter((other) => other !== id);
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

<!-- The balances sit above a list that scrolls on its own, so ticking a row
     shows its effect without scrolling away (WM-CAP-26). -->
<BalancePanel {projections} {reconciliation} {verb} />

<Card
  title="Suggested transactions"
  subtitle="{rows.length} found · {selectedCount} selected{committedCount
    ? ` · ${committedCount} written`
    : ''}. Highlighted fields need a closer look; click a confidence to see why."
>
  {#snippet actions()}
    <Button size="sm" disabled={busy} onclick={() => selectAll(true)}>Select all</Button>
    <Button size="sm" disabled={busy} onclick={() => selectAll(false)}>None</Button>
  {/snippet}

  {#if rows.length}
    <div class="-mx-5 max-h-[60vh] overflow-auto">
      <table class="w-full min-w-[68rem] text-sm">
        <thead class="sticky top-0 z-10 bg-white dark:bg-slate-900">
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
            disabled={locked(row)}
            problems={problems[index]}
            usdPreview={usdPreview(row)}
            balanceAfter={balancesAfter[index]}
            balanceCurrency={accountByName.get(row.tx.accountName)?.currency ?? 'VND'}
            {itemById}
            {activeAccounts}
            {categoryNames}
            {grouped}
            {tags}
            {cardTags}
            {verb}
            actionsDisabled={Boolean(pending)}
            dirty={isRowDirty(row)}
            saving={saving.includes(row.suggestion.id)}
            onapprove={() => commit([index])}
            onreject={reject ? () => rejectRows([row]) : undefined}
            onsave={save ? () => saveRow(row) : undefined}
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
      <Button
        icon="x"
        busy={rejecting}
        busyText="Rejecting…"
        disabled={committing || selectedCount === 0 || Boolean(pending)}
        onclick={rejectSelected}
      >
        Reject {selectedCount}
      </Button>
    {/if}
    <Button
      variant="primary"
      icon="check"
      busy={committing && !committingRow}
      busyText="Writing…"
      disabled={rejecting || (!pending && selectedCount === 0)}
      onclick={() => commit()}
    >
      {pending ? `Retry ${verb.toLowerCase()}` : `${verb} ${selectedCount} row${selectedCount === 1 ? '' : 's'}`}
    </Button>
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
