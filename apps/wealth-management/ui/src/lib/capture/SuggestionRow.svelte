<script lang="ts">
  import type { Account, CaptureField, Category } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { maskText } from '$lib/ai/moneyMask';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import { mask } from '$lib/state/mask.svelte';
  import { formatDate } from '$lib/format';
  import TagPicker from '$lib/transactions/TagPicker.svelte';
  import type { InboxItem } from './materials';
  import { confidenceLevel, type ReviewRow } from './review';

  // One suggested transaction: editable fields, evidence and write status.
  interface Props {
    row: ReviewRow;
    disabled: boolean;
    problems: string[];
    /** VND equivalent of a USD or USDT row, or null when not applicable. */
    usdPreview: number | null;
    itemById: Map<string, InboxItem>;
    activeAccounts: Account[];
    categoryNames: string[];
    grouped: { type: Category['type']; items: Category[] }[];
    tags: string[];
  }

  let {
    row = $bindable(),
    disabled,
    problems,
    usdPreview,
    itemById,
    activeAccounts,
    categoryNames,
    grouped,
    tags,
  }: Props = $props();

  const suggestion = $derived(row.suggestion);
  const level = $derived(confidenceLevel(suggestion.confidence));

  const low = (field: CaptureField) => (suggestion.lowConfidence.includes(field) ? 'ring-2 ring-amber-400/70' : '');
  const confidenceTone = { high: 'good', medium: 'info', low: 'warning' } as const;

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
      <select class="field w-48 {low('account')}" bind:value={row.tx.accountName} {disabled}>
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
      <input class="field w-48" bind:value={row.tx.payee} {disabled} />
    </td>
    <td class="px-2 pt-2">
      <select class="field w-44 {low('category')}" bind:value={row.tx.category} {disabled}>
        {#if !categoryNames.includes(row.tx.category)}<option value={row.tx.category}
            >{row.tx.category || 'Pick…'}</option
          >{/if}
        {#each grouped as group (group.type)}
          <optgroup label={group.type}>
            {#each group.items as category (category.name)}<option value={category.name}>{category.name}</option>{/each}
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

      {#if suggestion.duplicate}
        <p class="mt-2 text-xs text-amber-700 dark:text-amber-300">
          Likely already recorded: {suggestion.duplicate.payee} on {formatDate(suggestion.duplicate.date)},
          <Money value={suggestion.duplicate.amount} /> ({suggestion.duplicate.transactionId}). Unselected by default.
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
      {:else if row.error}
        <p class="mt-1 text-xs text-rose-600 dark:text-rose-400">Not written: {row.error}</p>
      {/if}
    </td>
  </tr>
</tbody>
