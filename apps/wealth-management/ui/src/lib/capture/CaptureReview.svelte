<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { Account, CaptureField, CaptureResult, CaptureRule, Category, TransactionInput } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import TagPicker from '$lib/transactions/TagPicker.svelte';
  import type { InboxItem } from './materials';
  import { commitBatch, confidenceLevel, learnedRules, reconcile, rowProblems, type ReviewRow } from './review';

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
  }

  let { rows = $bindable(), result, items, accounts, categories, tags, usdRate = null }: Props = $props();

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
  const pendingIndices = $derived(new Set(pending?.indices ?? []));
  const locked = (index: number) => Boolean(rows[index].createdId) || pendingIndices.has(index) || committing;

  function selectAll(selected: boolean) {
    for (const row of rows) if (!row.createdId) row.selected = selected;
  }

  async function commit() {
    let batch = pending;
    if (!batch) {
      if (problems.some((list) => list.length)) {
        toasts.push('warning', 'Fix the highlighted rows before committing.');
        return;
      }
      batch = { batchId: crypto.randomUUID(), ...commitBatch(rows) };
      if (!batch.inputs.length) return;
    }
    pending = batch;
    committing = true;
    try {
      const outcome = await api.createTransactions(batch.batchId, batch.inputs);
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
        toasts.push('warning', `${created} written, ${failed} failed. Fix the rows marked in red and commit again.`);
      else toasts.push('success', `${created} transaction${created === 1 ? '' : 's'} written to the sheet.`);
      await invalidateAll();
    } catch (error) {
      // A 4xx means the engine rejected the whole batch and wrote nothing.
      if (error instanceof ApiError && error.status >= 400 && error.status < 500) pending = null;
      toasts.error(error, pending ? 'No response from the engine; retry is safe' : 'Commit failed');
    } finally {
      committing = false;
    }
  }

  // Learned rules (WM-CAP-11): offered after rows are written.
  const ruleKey = (rule: CaptureRule) => `${rule.kind}:${rule.pattern.toLowerCase()}=${rule.value}`;
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

  const low = (index: number, field: CaptureField) =>
    rows[index].suggestion.lowConfidence.includes(field) ? 'ring-2 ring-amber-400/70' : '';
  const confidenceTone = { high: 'green', medium: 'blue', low: 'amber' } as const;

  /** VND equivalent of a USD or USDT row, rendered through Money so it masks (WM-UI-2). */
  function usdPreview(row: ReviewRow): number | null {
    const currency = accountByName.get(row.tx.accountName)?.currency;
    if (!usdRate || (currency !== 'USD' && currency !== 'USDT')) return null;
    const amount = row.tx.payment ?? row.tx.deposit ?? 0;
    return amount * usdRate;
  }

  function setAmount(row: ReviewRow, side: 'payment' | 'deposit', value: string) {
    const parsed = Number(value);
    row.tx[side] = value === '' || !Number.isFinite(parsed) ? undefined : parsed;
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
  <Card
    title="Balance check"
    subtitle="Balances shown in your materials vs the sheet after the selected rows. A difference hints at a missing or duplicated transaction; no adjustment is ever created."
    class="mb-4"
  >
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[40rem] text-sm">
        <thead>
          <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
            <th class="px-5 py-2 font-medium">Account</th>
            <th class="px-5 py-2 text-right font-medium">Shown in material</th>
            <th class="px-5 py-2 text-right font-medium">Sheet + selected</th>
            <th class="px-5 py-2 text-right font-medium">Difference</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each reconciliation as line (line.account)}
            <tr>
              <td class="px-5 py-2">
                {line.account}
                <p class="muted text-xs">as of {line.asOf} · {line.materialId}</p>
              </td>
              <td class="px-5 py-2 text-right"><Money value={line.balanceAfter} currency={line.currency} /></td>
              <td class="px-5 py-2 text-right">
                <Money value={line.projected} currency={line.currency} />
                <p class="muted text-xs">
                  <Money value={line.sheetBalance} currency={line.currency} /> + <Money
                    value={line.pending}
                    currency={line.currency}
                    signed
                  />
                </p>
              </td>
              <td class="px-5 py-2 text-right">
                {#if line.difference === 0}
                  <Badge tone="green">Matches</Badge>
                {:else}
                  <Money value={line.difference} currency={line.currency} signed tone />
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </Card>
{/if}

<Card
  title="Suggested transactions"
  subtitle="{rows.length} found · {selectedCount} selected{committedCount
    ? ` · ${committedCount} written`
    : ''}. Highlighted fields need a closer look."
>
  {#snippet actions()}
    <button type="button" class="btn-ghost" disabled={committing} onclick={() => selectAll(true)}>Select all</button>
    <button type="button" class="btn-ghost" disabled={committing} onclick={() => selectAll(false)}>None</button>
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
          {@const suggestion = row.suggestion}
          {@const disabled = locked(index)}
          {@const level = confidenceLevel(suggestion.confidence)}
          <tbody
            class="border-b border-slate-900/10 align-top dark:border-white/10 {row.selected || row.createdId
              ? ''
              : 'opacity-60'}"
          >
            <tr>
              <td class="px-3 pt-3">
                <input type="checkbox" aria-label="Select {row.tx.payee}" bind:checked={row.selected} {disabled} />
              </td>
              <td class="px-2 pt-2">
                <input type="date" class="field w-36 {low(index, 'date')}" bind:value={row.tx.date} {disabled} />
                {#if suggestion.occurredAt.length > 10}<p class="muted mt-0.5 text-xs">
                    {suggestion.occurredAt.slice(11)}
                  </p>{/if}
              </td>
              <td class="px-2 pt-2">
                <select class="field w-48 {low(index, 'account')}" bind:value={row.tx.accountName} {disabled}>
                  <option value="">Pick an account…</option>
                  {#if suggestion.accountCandidates?.length}
                    <optgroup label="Likely">
                      {#each suggestion.accountCandidates as name (name)}<option value={name}>{name}</option>{/each}
                    </optgroup>
                  {/if}
                  <optgroup label="All accounts">
                    {#each activeAccounts as account (account.name)}<option value={account.name}>{account.name}</option
                      >{/each}
                  </optgroup>
                </select>
                {#if suggestion.accountHint}<p class="muted mt-0.5 text-xs" title="Account shown in the material">
                    “{suggestion.accountHint}”
                  </p>{/if}
              </td>
              <td class="px-2 pt-2">
                <input class="field w-48" bind:value={row.tx.payee} {disabled} />
              </td>
              <td class="px-2 pt-2">
                <select class="field w-44 {low(index, 'category')}" bind:value={row.tx.category} {disabled}>
                  {#if !categoryNames.includes(row.tx.category)}<option value={row.tx.category}
                      >{row.tx.category || 'Pick…'}</option
                    >{/if}
                  {#each grouped as group (group.type)}
                    <optgroup label={group.type}>
                      {#each group.items as category (category.name)}<option value={category.name}
                          >{category.name}</option
                        >{/each}
                    </optgroup>
                  {/each}
                </select>
                {#if suggestion.categorySource}<p class="muted mt-0.5 text-xs">from {suggestion.categorySource}</p>{/if}
              </td>
              <td class="px-2 pt-2 text-right">
                <input
                  type="number"
                  min="0"
                  step="any"
                  class="field w-32 text-right {low(index, 'amount')}"
                  value={row.tx.payment ?? ''}
                  oninput={(event) => setAmount(row, 'payment', event.currentTarget.value)}
                  aria-label="Payment"
                  {disabled}
                />
              </td>
              <td class="px-2 pt-2 text-right">
                <input
                  type="number"
                  min="0"
                  step="any"
                  class="field w-32 text-right {low(index, 'amount')}"
                  value={row.tx.deposit ?? ''}
                  oninput={(event) => setAmount(row, 'deposit', event.currentTarget.value)}
                  aria-label="Deposit"
                  {disabled}
                />
                {#if usdPreview(row) !== null}<p class="muted mt-0.5 text-xs">
                    ≈ <Money value={usdPreview(row)} /> (sheet converts)
                  </p>{/if}
                {#if suggestion.currency !== 'VND'}<p class="muted mt-0.5 text-xs">
                    read as {suggestion.currency}
                  </p>{/if}
              </td>
              <td class="px-3 pt-3">
                <Badge tone={confidenceTone[level]}>{Math.round(suggestion.confidence * 100)}%</Badge>
              </td>
            </tr>
            <tr>
              <td></td>
              <td colspan="7" class="px-2 pt-2 pb-3">
                <div class="grid gap-2 md:grid-cols-[1fr_1fr_12rem]">
                  <div>
                    {#if disabled}
                      <p class="text-xs">{row.tx.tags.join(', ') || 'No tags'}</p>
                    {:else}
                      <TagPicker
                        options={[...new Set([...(suggestion.tagOptions ?? []), ...tags])]}
                        bind:selected={row.tx.tags}
                      />
                    {/if}
                    {#if suggestion.tagOptions?.length}<p class="muted mt-0.5 text-xs">
                        Card tags: {suggestion.tagOptions.join(', ')}
                      </p>{/if}
                  </div>
                  <input class="field" placeholder="Memo" aria-label="Memo" bind:value={row.tx.memo} {disabled} />
                  <input
                    class="field"
                    placeholder="Reference"
                    aria-label="Reference"
                    bind:value={row.tx.referenceNumber}
                    {disabled}
                  />
                </div>

                <div class="mt-2 flex flex-wrap items-start gap-2 text-xs">
                  {#if suggestion.transferGroup}<Badge tone="violet">Transfer {suggestion.transferGroup}</Badge>{/if}
                  {#if suggestion.sourceApp}<Badge>{suggestion.sourceApp}</Badge>{/if}
                  {#each suggestion.evidence as evidence (evidence.materialId)}
                    {@const item = itemById.get(evidence.materialId)}
                    <span
                      class="inline-flex max-w-md items-center gap-1.5 rounded-lg bg-slate-900/5 px-2 py-1 dark:bg-white/5"
                    >
                      {#if item?.previewUrl}
                        <a href={item.previewUrl} target="_blank" rel="noreferrer" title="Open {item.name}">
                          <img src={item.previewUrl} alt={item.name} class="h-8 w-8 rounded object-cover object-top" />
                        </a>
                      {/if}
                      <span class="font-medium">{evidence.materialId}</span>
                      {#if evidence.quote}<span class="muted truncate font-mono" title={evidence.quote}
                          >“{evidence.quote}”</span
                        >{/if}
                    </span>
                  {/each}
                </div>

                {#if suggestion.duplicate}
                  <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
                    Likely already recorded: {suggestion.duplicate.payee} on {formatDate(suggestion.duplicate.date)},
                    <Money value={suggestion.duplicate.amount} /> ({suggestion.duplicate.transactionId}). Unselected by
                    default.
                  </p>
                {/if}
                {#each suggestion.notes ?? [] as note (note)}<p class="muted mt-1 text-xs">{note}</p>{/each}
                {#each problems[index] as problem (problem)}
                  <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">{problem}</p>
                {/each}
                {#if row.createdId}
                  <p class="mt-1 inline-flex items-center gap-1 text-xs text-emerald-700 dark:text-emerald-300">
                    <Icon name="check" size={14} /> Written as {row.createdId}
                  </p>
                {:else if row.error}
                  <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">Not written: {row.error}</p>
                {/if}
              </td>
            </tr>
          </tbody>
        {/each}
      </table>
    </div>
  {:else}
    <p class="muted text-sm">No transactions were found in these materials.</p>
  {/if}

  <div class="mt-4 flex flex-wrap items-center justify-end gap-3">
    {#if pending && !committing}
      <p class="muted text-xs">
        The last commit got no response. Retrying sends the same rows and cannot write them twice.
      </p>
    {/if}
    <button
      type="button"
      class="btn-primary"
      disabled={committing || (!pending && selectedCount === 0)}
      onclick={commit}
    >
      {committing
        ? 'Writing…'
        : pending
          ? 'Retry commit'
          : `Commit ${selectedCount} row${selectedCount === 1 ? '' : 's'}`}
    </button>
  </div>
</Card>

{#if rules.length}
  <Card
    title="Remember for next time"
    subtitle="Saved to the CaptureRules tab and used by later captures."
    class="mt-4"
  >
    <ul class="space-y-1 text-sm">
      {#each rules as rule (ruleKey(rule))}
        <li>
          <label class="inline-flex items-center gap-2">
            <input
              type="checkbox"
              checked={!skippedRules.includes(ruleKey(rule))}
              onchange={(event) =>
                (skippedRules = event.currentTarget.checked
                  ? skippedRules.filter((key) => key !== ruleKey(rule))
                  : [...skippedRules, ruleKey(rule)])}
            />
            {#if rule.kind === 'alias'}
              “{rule.pattern}” is account <strong>{rule.value}</strong>
            {:else}
              Payee “{rule.pattern}” is category <strong>{rule.value}</strong>
            {/if}
          </label>
        </li>
      {/each}
    </ul>
    <div class="mt-3 flex justify-end">
      <button type="button" class="btn-ghost" disabled={savingRules || !rulesToSave.length} onclick={saveRules}>
        {savingRules ? 'Saving…' : `Save ${rulesToSave.length} rule${rulesToSave.length === 1 ? '' : 's'}`}
      </button>
    </div>
  </Card>
{/if}
