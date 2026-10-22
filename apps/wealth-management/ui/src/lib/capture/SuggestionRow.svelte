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
  import TagPicker from '$lib/transactions/TagPicker.svelte';
  import ConfidenceBadge from './ConfidenceBadge.svelte';
  import type { InboxItem } from './materials';
  import {
    applyCategorySuggestion,
    fillCardTag,
    nonCardTags,
    pickCardTag,
    pickedCardTag,
    type ReviewRow,
    type RunningBalance,
  } from './review';

  // One suggested transaction: editable fields, evidence, write status and
  // its own approve, reject and save actions.
  interface Props {
    row: ReviewRow;
    disabled: boolean;
    problems: string[];
    /** VND equivalent of a USD or USDT row, or null when not applicable. */
    usdPreview: number | null;
    /** What the account has once this row and the selected rows above it are written; null when not projected. */
    balanceAfter: RunningBalance | null;
    balanceCurrency: string;
    itemById: Map<string, InboxItem>;
    activeAccounts: Account[];
    categoryNames: string[];
    grouped: { type: Category['type']; items: Category[] }[];
    tags: string[];
    /** Open card keywords per account: the tags saying which card paid. */
    cardTags: Map<string, string[]>;
    /** Open card names per account by lowercased keyword. */
    cardNames: Map<string, Map<string, string>>;
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
    cardNames,
    verb,
    actionsDisabled,
    dirty,
    saving,
    onapprove,
    onreject,
    onsave,
  }: Props = $props();

  const suggestion = $derived(row.suggestion);
  // The note box opens on demand; a transfer or fee row has none (WM-CAP-24).
  const notable = $derived(!suggestion.transferGroup && !suggestion.fee);
  let noteOpen = $state(false);
  // Opening the note box puts the caret at the end of the note.
  function focusNote(node: HTMLTextAreaElement) {
    node.focus();
    node.setSelectionRange(node.value.length, node.value.length);
  }

  const low = (field: CaptureField) => (suggestion.lowConfidence.includes(field) ? 'ring-2 ring-amber-400/70' : '');

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
  // the tag picker shows the card tag but edits only the user's tags.
  const userTags = $derived(nonCardTags(row.tx.tags, cardTags));
  const cardTag = $derived(pickedCardTag(row.tx, accountCardTags));
  // The card is always shown: a picker with several open cards, its name with one.
  const cardName = (keyword: string) => cardNames.get(row.tx.accountName)?.get(keyword.toLowerCase()) ?? keyword;

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
  class="border-b border-stone-900/10 align-top dark:border-white/10 {row.selected || row.createdId
    ? ''
    : 'opacity-60'}"
>
  <tr>
    <td class="px-3 pt-3">
      <input type="checkbox" aria-label="Select {row.tx.payee}" bind:checked={row.selected} {disabled} />
    </td>
    <td class="px-2 pt-2">
      <input type="date" class="field w-36 {low('date')}" aria-label="Date" bind:value={row.tx.date} {disabled} />
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
            class="link hit-area inline-flex items-center gap-0.5"
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
      {#if balanceAfter?.credit}<p
          class="muted mt-0.5 text-xs"
          title="Credit left on the {row.tx.accountName} limit after this row"
        >
          Credit left after: <Money value={balanceAfter.amount} currency={balanceCurrency} />
        </p>
      {:else if balanceAfter}<p class="muted mt-0.5 text-xs" title="{row.tx.accountName} balance after this row">
          Balance after: <Money value={balanceAfter.amount} currency={balanceCurrency} />
        </p>{/if}
    </td>
    <td class="px-3 pt-3">
      <ConfidenceBadge overall={suggestion.overall} reasons={suggestion.reasons} notes={suggestion.notes} />
    </td>
  </tr>
  <tr>
    <td></td>
    <td colspan="7" class="px-2 pt-2 pb-3">
      <div class="grid grid-cols-1 gap-2 md:grid-cols-[1fr_1fr_12rem]">
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
                {#each accountCardTags as option (option)}<option value={option}>{cardName(option)}</option>{/each}
              </select>
            {:else if cardTag}
              <p class="mb-2 text-xs">Card: {cardName(cardTag)}</p>
            {/if}
            <TagPicker
              fixed={row.tx.tags
                .filter((tag) => !userTags.includes(tag))
                .map((tag) => ({ tag, title: `${tag}: the card that paid, not edited here` }))}
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
      {#if notable && noteOpen}
        <textarea
          class="field mt-2 w-full"
          rows="1"
          maxlength="500"
          placeholder="Note for next time: what this payment is for. Kept by the app, not written to the sheet."
          aria-label="Note for next time"
          bind:value={row.note}
          {disabled}
          {@attach focusNote}
        ></textarea>
        {#if suggestion.pastNotes?.length}
          <p class="muted mt-1 text-xs">Your earlier notes: {suggestion.pastNotes.join(' · ')}</p>
        {/if}
      {/if}

      <div class="mt-2 flex flex-wrap items-start gap-2 text-xs">
        {#if suggestion.transferGroup}<Badge tone="violet">Transfer {suggestion.transferGroup}</Badge>{/if}
        {#if suggestion.sourceApp}<Badge>{suggestion.sourceApp}</Badge>{/if}
        {#each suggestion.evidence as evidence (evidence.materialId)}
          {@const item = itemById.get(evidence.materialId)}
          <span class="inline-flex max-w-md items-center gap-1.5 rounded-lg bg-stone-900/5 px-2 py-1 dark:bg-white/5">
            {#if item?.previewUrl}
              <a
                href={item.previewUrl}
                target="_blank"
                rel="noreferrer"
                title="Open {item.name}"
                class="block rounded transition hover:ring-2 hover:ring-accent-400/60 active:opacity-80"
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
          {#if notable}
            <Button
              size="sm"
              icon="pencil"
              class="mr-auto"
              aria-expanded={noteOpen}
              onclick={() => (noteOpen = !noteOpen)}>{row.note?.trim() ? 'Edit note' : 'Add note'}</Button
            >
          {/if}
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
