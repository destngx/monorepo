<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Account, Category, CategorySuggestion, Transaction, TransactionInput } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
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
  // After a rejected submit the fields at fault are marked invalid until fixed.
  let attempted = $state(false);
  const missing = $derived({
    accountName: attempted && !form.accountName,
    date: attempted && !form.date,
    payee: attempted && !form.payee.trim(),
    category: attempted && !form.category,
  });
  const amountInvalid = $derived.by(() => {
    if (!attempted) return false;
    const payment = Number(form.payment) || 0;
    const deposit = Number(form.deposit) || 0;
    return payment < 0 || deposit < 0 || (payment > 0 && deposit > 0) || (payment === 0 && deposit === 0);
  });

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
    attempted = true;
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
    attempted = false;
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
    <select
      id="tx-account"
      class="field"
      bind:value={form.accountName}
      required
      aria-invalid={missing.accountName || undefined}
    >
      <option value="" disabled>Choose an account</option>
      {#each activeAccounts as account (account.name)}
        <option value={account.name}>{account.name}</option>
      {/each}
    </select>
  </div>
  <div>
    <label class="label" for="tx-date">Date *</label>
    <input
      id="tx-date"
      type="date"
      class="field"
      bind:value={form.date}
      required
      aria-invalid={missing.date || undefined}
    />
  </div>
  <div class="sm:col-span-2">
    <label class="label" for="tx-payee">Payee *</label>
    <input
      id="tx-payee"
      class="field"
      bind:value={form.payee}
      required
      autocomplete="off"
      aria-invalid={missing.payee || undefined}
    />
  </div>
  <div class="sm:col-span-2">
    <div class="flex items-center justify-between gap-2">
      <label class="label" for="tx-category">Category *</label>
      <Button
        size="sm"
        icon="sparkles"
        class="-mt-1"
        busy={suggesting}
        busyText="Suggesting…"
        disabled={[...form.payee.trim()].length < SUGGEST_MIN_CHARS}
        onclick={suggestNow}
        title="Suggest a category from the payee">Suggest</Button
      >
    </div>
    <select
      id="tx-category"
      class="field"
      bind:value={form.category}
      required
      aria-invalid={missing.category || undefined}
    >
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
    <input
      id="tx-payment"
      type="number"
      min="0"
      step="1"
      inputmode="numeric"
      class="field"
      bind:value={form.payment}
      aria-invalid={amountInvalid || undefined}
    />
  </div>
  <div>
    <label class="label" for="tx-deposit">Deposit (money in)</label>
    <input
      id="tx-deposit"
      type="number"
      min="0"
      step="1"
      inputmode="numeric"
      class="field"
      bind:value={form.deposit}
      aria-invalid={amountInvalid || undefined}
    />
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
    <Button disabled={busy} onclick={oncancel}>Cancel</Button>
    {#if !initial}
      <Button type="submit" value="another" disabled={busy}>Save & add another</Button>
    {/if}
    <Button type="submit" variant="primary" {busy} busyText="Saving…"
      >{initial ? 'Save changes' : 'Add transaction'}</Button
    >
  </div>
</form>
