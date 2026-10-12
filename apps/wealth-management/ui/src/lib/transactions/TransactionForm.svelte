<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Account, Category, CategorySuggestion, Transaction, TransactionInput } from '$lib/api/types';
  import Icon from '$lib/components/Icon.svelte';
  import { todayISO } from '$lib/format';
  import TagPicker from './TagPicker.svelte';

  interface Props {
    accounts: Account[];
    categories: Category[];
    tags: string[];
    /** Existing row when editing. */
    initial?: Transaction | null;
    defaultAccount?: string;
    /** Seed values for a new row (deep links, "Save & add another"). */
    prefill?: Partial<TransactionInput>;
    busy?: boolean;
    /** `addAnother` is set by "Save & add another" (WM-TX-9). */
    onsubmit: (...args: [input: TransactionInput, addAnother: boolean]) => void;
    oncancel: () => void;
  }

  let {
    accounts,
    categories,
    tags,
    initial = null,
    defaultAccount = '',
    prefill = {},
    busy = false,
    onsubmit,
    oncancel,
  }: Props = $props();

  // The form edits a local copy; props only seed it.
  const seed = (): TransactionInput => ({
    accountName: initial?.accountName ?? (prefill.accountName || defaultAccount),
    date: initial?.date.slice(0, 10) ?? (prefill.date || todayISO()),
    payee: initial?.payee ?? prefill.payee ?? '',
    category: initial?.category ?? prefill.category ?? '',
    tags: [...(initial?.tags ?? [])],
    memo: initial?.memo ?? '',
    referenceNumber: initial?.referenceNumber ?? '',
    cleared: initial?.cleared ?? prefill.cleared ?? true,
    payment: initial?.payment ?? undefined,
    deposit: initial?.deposit ?? undefined,
  });
  let form = $state(seed());
  let error = $state<string | null>(null);

  const activeAccounts = $derived(accounts.filter((a) => a.type !== 'deprecated' || a.name === form.accountName));
  const grouped = $derived(
    (['expense', 'income', 'non-budget'] as const).map((type) => ({
      type,
      items: categories.filter((c) => c.type === type),
    })),
  );

  // AI category suggestion (WM-TX-5): automatic only while the category is
  // empty; the engine guarantees the result is an existing category.
  const SUGGEST_MIN_CHARS = 3;
  const SUGGEST_DEBOUNCE_MS = 1000;
  let suggesting = $state(false);
  let suggestion = $state<CategorySuggestion | null>(null);
  let suggestController: AbortController | null = null;

  async function suggest(payee: string, auto: boolean) {
    suggestController?.abort();
    const controller = new AbortController();
    suggestController = controller;
    suggesting = true;
    try {
      const result = await api.suggestCategory(payee, controller.signal);
      if (controller.signal.aborted) return;
      suggestion = result;
      // An automatic suggestion never overrides a category the user picked meanwhile.
      const canApply = !auto || !form.category;
      if (canApply && result.source !== 'default' && categories.some((c) => c.name === result.category)) {
        form.category = result.category;
      }
    } catch (failure) {
      if (!controller.signal.aborted) {
        suggestion = null;
        error =
          failure instanceof Error ? `Category suggestion failed: ${failure.message}` : 'Category suggestion failed.';
      }
    } finally {
      if (suggestController === controller) suggesting = false;
    }
  }

  function suggestNow() {
    const payee = form.payee.trim();
    if ([...payee].length >= SUGGEST_MIN_CHARS) void suggest(payee, false);
  }

  $effect(() => {
    const payee = form.payee.trim();
    if (initial || form.category || [...payee].length < SUGGEST_MIN_CHARS) return;
    const timer = setTimeout(() => void suggest(payee, true), SUGGEST_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  });

  $effect(() => () => suggestController?.abort());

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const addAnother = (event.submitter as HTMLButtonElement | null)?.value === 'another';
    const payment = Number(form.payment) || 0;
    const deposit = Number(form.deposit) || 0;
    if (!form.accountName || !form.date || !form.payee.trim() || !form.category) {
      error = 'Account, date, payee and category are required.';
      return;
    }
    if (payment < 0 || deposit < 0) {
      error = 'Amounts must be positive; use payment for money out and deposit for money in.';
      return;
    }
    if (payment > 0 && deposit > 0) {
      error = 'Enter either a payment or a deposit, not both.';
      return;
    }
    if (payment === 0 && deposit === 0) {
      error = 'Enter a payment or a deposit amount.';
      return;
    }
    error = null;
    onsubmit(
      {
        ...form,
        payee: form.payee.trim(),
        memo: form.memo?.trim() || undefined,
        referenceNumber: form.referenceNumber?.trim() || undefined,
        payment: payment || undefined,
        deposit: deposit || undefined,
      },
      addAnother,
    );
  }
</script>

<form id="transaction-form" class="grid gap-4 sm:grid-cols-2" onsubmit={submit} novalidate>
  <div>
    <label class="label" for="tx-account">Account *</label>
    <select id="tx-account" class="field" bind:value={form.accountName} required>
      <option value="" disabled>Choose an account</option>
      {#each activeAccounts as account (account.name)}
        <option value={account.name}>{account.name}</option>
      {/each}
    </select>
  </div>
  <div>
    <label class="label" for="tx-date">Date *</label>
    <input id="tx-date" type="date" class="field" bind:value={form.date} required />
  </div>
  <div class="sm:col-span-2">
    <label class="label" for="tx-payee">Payee *</label>
    <input id="tx-payee" class="field" bind:value={form.payee} required autocomplete="off" />
  </div>
  <div class="sm:col-span-2">
    <div class="flex items-center justify-between gap-2">
      <label class="label" for="tx-category">Category *</label>
      <button
        type="button"
        class="btn-ghost -mt-1 px-2 py-0.5 text-xs"
        disabled={suggesting || [...form.payee.trim()].length < SUGGEST_MIN_CHARS}
        onclick={suggestNow}
        title="Suggest a category from the payee"
      >
        <Icon name="sparkles" size={14} />
        {suggesting ? 'Suggesting…' : 'Suggest'}
      </button>
    </div>
    <select id="tx-category" class="field" bind:value={form.category} required>
      <option value="" disabled>Choose a category</option>
      {#each grouped as group (group.type)}
        {#if group.items.length}
          <optgroup label={group.type}>
            {#each group.items as category (category.name)}
              <option value={category.name}>{category.name}</option>
            {/each}
          </optgroup>
        {/if}
      {/each}
    </select>
    {#if suggestion}
      <p class="muted mt-1 text-xs" aria-live="polite">
        {#if suggestion.source === 'default'}
          No confident suggestion for this payee.
        {:else}
          Suggested <span class="font-medium">{suggestion.category}</span>
          {suggestion.source === 'history' ? 'from your history' : 'by AI'}.
        {/if}
      </p>
    {/if}
  </div>
  <div>
    <label class="label" for="tx-payment">Payment (money out)</label>
    <input id="tx-payment" type="number" min="0" step="1" inputmode="numeric" class="field" bind:value={form.payment} />
  </div>
  <div>
    <label class="label" for="tx-deposit">Deposit (money in)</label>
    <input id="tx-deposit" type="number" min="0" step="1" inputmode="numeric" class="field" bind:value={form.deposit} />
  </div>
  <div class="sm:col-span-2">
    <span class="label">Tags</span>
    <TagPicker options={tags} bind:selected={form.tags} />
  </div>
  <div class="sm:col-span-2">
    <label class="label" for="tx-memo">Memo</label>
    <input id="tx-memo" class="field" bind:value={form.memo} />
  </div>
  <div>
    <label class="label" for="tx-ref">Reference</label>
    <input id="tx-ref" class="field" bind:value={form.referenceNumber} />
  </div>
  <label class="mt-6 flex items-center gap-2 text-sm">
    <input type="checkbox" bind:checked={form.cleared} class="size-4 accent-indigo-600" /> Cleared
  </label>
  {#if error}<p role="alert" class="text-sm text-orange-600 sm:col-span-2 dark:text-orange-400">{error}</p>{/if}
  <div class="flex justify-end gap-2 sm:col-span-2">
    <button type="button" class="btn-ghost" onclick={oncancel}>Cancel</button>
    {#if !initial}
      <button type="submit" class="btn-ghost" value="another" disabled={busy}>Save & add another</button>
    {/if}
    <button type="submit" class="btn-primary" disabled={busy}
      >{busy ? 'Saving…' : initial ? 'Save changes' : 'Add transaction'}</button
    >
  </div>
</form>
