<script lang="ts">
  import { untrack } from 'svelte';
  import { api } from '$lib/api/endpoints';
  import type { Account, CaptureField, Category } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { maskText } from '$lib/ai/moneyMask';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import { mask } from '$lib/state/mask.svelte';
  import { formatDate } from '$lib/format';
  import TagPicker from '$lib/transactions/TagPicker.svelte';
  import type { InboxItem } from './materials';
  import {
    applyCategorySuggestion,
    confidenceLevel,
    fillCardTag,
    nonCardTags,
    pickCardTag,
    pickedCardTag,
    type ReviewRow,
  } from './review';

  // One suggested transaction: editable fields, evidence, write status and
  // its own approve, reject and save actions.
  interface Props {
    row: ReviewRow;
    disabled: boolean;
    problems: string[];
    /** VND equivalent of a USD or USDT row, or null when not applicable. */
    usdPreview: number | null;
    /** The account balance once this row and the selected rows above it are written; null when not projected. */
    balanceAfter: number | null;
    balanceCurrency: string;
    itemById: Map<string, InboxItem>;
    activeAccounts: Account[];
    categoryNames: string[];
    grouped: { type: Category['type']; items: Category[] }[];
    tags: string[];
    /** Open card keywords per account: the tags saying which card paid. */
    cardTags: Map<string, string[]>;
    /** Verb of the write button: "Commit" or "Approve". */
    verb: string;
    /** True while an unanswered batch waits for its retry. */
    actionsDisabled: boolean;
    /** The row has edits not saved yet (drafts only). */
    dirty: boolean;
    saving: boolean;
    onapprove: () => Promise<void>;
    onreject?: () => Promise<void>;
    onsave?: () => Promise<void>;
  }

  let {
    row = $bindable(),
    disabled,
    problems,
    usdPreview,
    balanceAfter,
    balanceCurrency,
    itemById,
    activeAccounts,
    categoryNames,
    grouped,
    tags,
    cardTags,
    verb,
    actionsDisabled,
    dirty,
    saving,
    onapprove,
    onreject,
    onsave,
  }: Props = $props();

  const suggestion = $derived(row.suggestion);
  const level = $derived(confidenceLevel(suggestion.confidence));

  const low = (field: CaptureField) => (suggestion.lowConfidence.includes(field) ? 'ring-2 ring-amber-400/70' : '');
  const confidenceTone = { high: 'good', medium: 'info', low: 'warning' } as const;

  // A new payee gets a new category suggestion (WM-TX-5 rules: 3 chars,
  // 1 s debounce): automatic unless the user picked the category by hand.
  const SUGGEST_MIN_CHARS = 3;
  const SUGGEST_DEBOUNCE_MS = 1000;
  let suggestedFor = untrack(() => row.tx.payee.trim());
  let suggesting = $state(false);
  let suggestError = $state('');
  let suggestController: AbortController | null = null;

  async function suggest(payee: string, force: boolean) {
    suggestController?.abort();
    const controller = new AbortController();
    suggestController = controller;
    suggesting = true;
    suggestError = '';
    try {
      const result = await api.suggestCategory(payee, controller.signal);
      if (controller.signal.aborted || row.tx.payee.trim() !== payee) return;
      suggestedFor = payee;
      applyCategorySuggestion(row, result, categoryNames, { force });
    } catch (failure) {
      if (!controller.signal.aborted) {
        suggestError = failure instanceof Error ? failure.message : 'unknown error';
      }
    } finally {
      if (suggestController === controller) suggesting = false;
    }
  }

  // The account's open cards; the engine's options are for the account it matched.
  const accountCardTags = $derived(
    cardTags.get(row.tx.accountName) ??
      (row.tx.accountName === suggestion.transaction.accountName ? (suggestion.tagOptions ?? []) : []),
  );

  // The card is picked on its own where the account has several open cards;
  // the tag picker edits the user's tags and leaves the card tag in place.
  const userTags = $derived(nonCardTags(row.tx.tags, cardTags));
  const cardTag = $derived(pickedCardTag(row.tx, accountCardTags));

  const canSuggest = $derived(!disabled && !suggestion.transferGroup);

  function suggestNow() {
    const payee = row.tx.payee.trim();
    if ([...payee].length >= SUGGEST_MIN_CHARS) void suggest(payee, true);
  }

  $effect(() => {
    const payee = row.tx.payee.trim();
    if (!canSuggest || row.categorySource === 'manual' || payee === suggestedFor) return;
    if ([...payee].length < SUGGEST_MIN_CHARS) return;
    const timer = setTimeout(() => void suggest(payee, false), SUGGEST_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  });

  $effect(() => () => suggestController?.abort());

  function setAmount(side: 'payment' | 'deposit', value: string) {
    const parsed = Number(value);
    row.tx[side] = value === '' || !Number.isFinite(parsed) ? undefined : parsed;
  }
</script>

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
      <input type="date" class="field w-36 {low('date')}" bind:value={row.tx.date} {disabled} />
      {#if suggestion.occurredAt.length > 10}<p class="muted mt-0.5 text-xs">
          {suggestion.occurredAt.slice(11)}
        </p>{/if}
    </td>
    <td class="px-2 pt-2">
      <select
        class="field w-48 {low('account')}"
        aria-label="Account"
        bind:value={row.tx.accountName}
        onchange={() => fillCardTag(row.tx, cardTags)}
        {disabled}
      >
        <option value="">Pick an account…</option>
        {#if suggestion.accountCandidates?.length}
          <optgroup label="Likely">
            {#each suggestion.accountCandidates as name (name)}<option value={name}>{name}</option>{/each}
          </optgroup>
        {/if}
        <optgroup label="All accounts">
          {#each activeAccounts as account (account.name)}<option value={account.name}>{account.name}</option>{/each}
        </optgroup>
      </select>
      {#if suggestion.accountHint}<p class="muted mt-0.5 text-xs" title="Account shown in the material">
          “{suggestion.accountHint}”
        </p>{/if}
    </td>
    <td class="px-2 pt-2">
      <input class="field w-48 {low('payee')}" aria-label="Payee" bind:value={row.tx.payee} {disabled} />
    </td>
    <td class="px-2 pt-2">
      <select
        class="field w-44 {low('category')}"
        aria-label="Category"
        bind:value={row.tx.category}
        onchange={() => (row.categorySource = 'manual')}
        {disabled}
      >
        {#if !categoryNames.includes(row.tx.category)}<option value={row.tx.category}
            >{row.tx.category || 'Pick…'}</option
          >{/if}
        {#each grouped as group (group.type)}
          <optgroup label={group.type}>
            {#each group.items as category (category.name)}<option value={category.name}>{category.name}</option>{/each}
          </optgroup>
        {/each}
      </select>
      <p class="muted mt-0.5 flex items-center gap-1.5 text-xs" aria-live="polite">
        {#if suggesting}
          Suggesting…
        {:else if row.categorySource}
          {row.categorySource === 'manual' ? 'picked by you' : `from ${row.categorySource}`}
        {/if}
        {#if canSuggest && !suggesting}
          <button
            type="button"
            class="link inline-flex items-center gap-0.5"
            title="Suggest a category for this payee"
            onclick={suggestNow}><Icon name="sparkles" size={12} /> Suggest</button
          >
        {/if}
      </p>
      {#if suggestError}<p class="mt-0.5 text-xs text-rose-600 dark:text-rose-400">
          Suggestion failed: {suggestError}
        </p>{/if}
    </td>
    <td class="px-2 pt-2 text-right">
      <input
        type="number"
        min="0"
        step="any"
        class="field w-32 text-right {low('amount')}"
        value={row.tx.payment ?? ''}
        oninput={(event) => setAmount('payment', event.currentTarget.value)}
        aria-label="Payment"
        {disabled}
      />
    </td>
    <td class="px-2 pt-2 text-right">
      <input
        type="number"
        min="0"
        step="any"
        class="field w-32 text-right {low('amount')}"
        value={row.tx.deposit ?? ''}
        oninput={(event) => setAmount('deposit', event.currentTarget.value)}
        aria-label="Deposit"
        {disabled}
      />
      {#if usdPreview !== null}<p class="muted mt-0.5 text-xs">
          ≈ <Money value={usdPreview} /> (sheet converts)
        </p>{/if}
      {#if suggestion.currency !== 'VND'}<p class="muted mt-0.5 text-xs">
          read as {suggestion.currency}
        </p>{/if}
      {#if balanceAfter !== null}<p class="muted mt-0.5 text-xs" title="{row.tx.accountName} balance after this row">
          Balance after: <Money value={balanceAfter} currency={balanceCurrency} />
        </p>{/if}
    </td>
    <td class="px-3 pt-3">
      <Badge status={confidenceTone[level]}>{Math.round(suggestion.confidence * 100)}%</Badge>
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
            {#if accountCardTags.length > 1}
              <select
                class="field mb-2"
                aria-label="Card"
                value={cardTag}
                onchange={(event) => pickCardTag(row.tx, event.currentTarget.value, cardTags)}
              >
                <option value="" disabled>Pick the card…</option>
                {#each accountCardTags as option (option)}<option value={option}>{option}</option>{/each}
              </select>
            {/if}
            <TagPicker
              options={nonCardTags([...new Set([...(suggestion.tagOptions ?? []), ...tags])], cardTags)}
              bind:selected={
                () => userTags,
                (own) => (row.tx.tags = [...row.tx.tags.filter((tag) => !userTags.includes(tag)), ...own])
              }
            />
          {/if}
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
      {#if !suggestion.transferGroup && !suggestion.fee}
        <textarea
          class="field mt-2 w-full"
          rows="1"
          maxlength="500"
          placeholder="Note for next time: what this payment is for. Kept by the app, not written to the sheet."
          aria-label="Note for next time"
          bind:value={row.note}
          {disabled}
        ></textarea>
      {/if}
      {#if suggestion.pastNotes?.length}
        <p class="muted mt-1 text-xs">Your earlier notes: {suggestion.pastNotes.join(' · ')}</p>
      {/if}

      <div class="mt-2 flex flex-wrap items-start gap-2 text-xs">
        {#if suggestion.transferGroup}<Badge tone="violet">Transfer {suggestion.transferGroup}</Badge>{/if}
        {#if suggestion.sourceApp}<Badge>{suggestion.sourceApp}</Badge>{/if}
        {#each suggestion.evidence as evidence (evidence.materialId)}
          {@const item = itemById.get(evidence.materialId)}
          <span class="inline-flex max-w-md items-center gap-1.5 rounded-lg bg-slate-900/5 px-2 py-1 dark:bg-white/5">
            {#if item?.previewUrl}
              <a
                href={item.previewUrl}
                target="_blank"
                rel="noreferrer"
                title="Open {item.name}"
                class="block rounded transition hover:ring-2 hover:ring-indigo-400/60 active:opacity-80"
              >
                <img src={item.previewUrl} alt={item.name} class="h-8 w-8 rounded object-cover object-top" />
              </a>
            {/if}
            <span class="font-medium">{item?.label ?? evidence.materialId}</span>
            {#if evidence.quote}<span class="muted truncate font-mono" title={maskText(evidence.quote, mask.revealed)}
                >“<MaskedText text={evidence.quote} />”</span
              >{/if}
          </span>
        {/each}
      </div>

      {#if !suggestion.selected && !suggestion.duplicate?.likely}
        <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
          Not selected: read with {Math.round(suggestion.confidence * 100)}% confidence. Check it, then tick it.
        </p>
      {/if}
      {#if suggestion.learnedPayee}
        {@const learned = suggestion.learnedPayee}
        {@const times = learned.count === 1 ? 'once' : `${learned.count} times`}
        <p class="muted mt-2 text-xs">
          Payee learned: {learned.counterparty}
          {learned.count
            ? `was written as ${suggestion.transaction.payee} ${times} before.`
            : `is ${suggestion.transaction.payee} in CaptureRules.`}
          {#if learned.others?.length}
            <span class="text-amber-700 dark:text-amber-300"
              >Also written as {learned.others.join(', ')}: check the payee.</span
            >
          {/if}
        </p>
      {/if}
      {#if suggestion.duplicate}
        {@const duplicate = suggestion.duplicate}
        {@const label = duplicate.likely
          ? `Likely already recorded (${duplicate.score}% match)`
          : `Possible match (${duplicate.score}%)`}
        {@const gap =
          duplicate.days === 0 ? 'same day' : `${duplicate.days} day${duplicate.days === 1 ? '' : 's'} apart`}
        <p class="mt-2 text-xs {duplicate.likely ? 'text-amber-700 dark:text-amber-300' : 'muted'}">
          {label}: {duplicate.payee} on {formatDate(duplicate.date)}, <Money value={duplicate.amount} /> ({duplicate.transactionId}),
          {gap}. {duplicate.likely
            ? 'Unselected by default.'
            : 'Still selected; untick it if it is the same transaction.'}
        </p>
      {:else if suggestion.recordedThrough}
        <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
          May be missing from the sheet: {suggestion.transaction.accountName} is recorded up to
          {formatDate(suggestion.recordedThrough)}, but no row there matches this one. Still selected.
        </p>
      {/if}
      {#each suggestion.notes ?? [] as note (note)}<p class="muted mt-1 text-xs"><MaskedText text={note} /></p>{/each}
      {#each problems as problem (problem)}
        <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">{problem}</p>
      {/each}
      {#if row.createdId}
        <p class="mt-1 inline-flex items-center gap-1 text-xs text-emerald-700 dark:text-emerald-300">
          <Icon name="check" size={14} /> Written as {row.createdId}
        </p>
      {:else}
        {#if row.error}
          <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">Not written: {row.error}</p>
        {/if}
        <div class="mt-2 flex flex-wrap items-center justify-end gap-2">
          {#if dirty}<span class="muted text-xs">Unsaved changes</span>{/if}
          {#if onsave}
            <Button
              size="sm"
              busy={saving}
              busyText="Saving…"
              disabled={(disabled && !saving) || actionsDisabled || !dirty}
              title={dirty ? 'Keep these edits without approving' : 'No changes to save'}
              onclick={onsave}>Save</Button
            >
          {/if}
          {#if onreject}
            <Button size="sm" icon="x" disabled={disabled || actionsDisabled} onclick={onreject}>Reject</Button>
          {/if}
          <Button
            size="sm"
            variant="primary"
            icon="check"
            busyText="Writing…"
            disabled={disabled || actionsDisabled}
            onclick={onapprove}>{verb}</Button
          >
        </div>
      {/if}
    </td>
  </tr>
</tbody>
