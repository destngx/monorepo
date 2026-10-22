<script lang="ts">
  import { base } from '$app/paths';
  import { adjustmentHref } from '$lib/accounts';
  import { ACCOUNT_TYPES, type Account } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import type { AccountForm } from './form';
  import type { SubmitHandler } from './handlers';

  // Create / edit (WM-ACC-6, WM-ACC-7).
  interface Props {
    open: boolean;
    /** The account being edited, or null when creating. */
    editing: Account | null;
    form: AccountForm;
    busy: boolean;
    onclose: () => void;
    onsubmit: SubmitHandler;
  }

  let { open, editing, form = $bindable(), busy, onclose, onsubmit }: Props = $props();

  // The name shows as invalid once the field was left or a submit was tried while it is blank.
  let touched = $state(false);
  const nameInvalid = $derived(!editing && touched && !form.name.trim());
  $effect(() => {
    if (!open) touched = false;
  });

  function submit(event: SubmitEvent) {
    touched = true;
    onsubmit(event);
  }
</script>

<Modal {open} title={editing ? `Edit ${editing.name}` : 'New account'} {onclose} {busy}>
  <form id="account-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={submit}>
    {#if !editing}
      <div class="sm:col-span-2">
        <label class="label" for="acc-name">Name *</label>
        <input
          id="acc-name"
          class="field"
          bind:value={form.name}
          required
          aria-invalid={nameInvalid}
          aria-describedby={nameInvalid ? 'acc-name-error' : undefined}
          onblur={() => (touched = true)}
          oninvalid={() => (touched = true)}
        />
        {#if nameInvalid}
          <p id="acc-name-error" class="mt-1 text-xs text-rose-600 dark:text-rose-400">Enter a name for the account.</p>
        {/if}
      </div>
    {/if}
    <div>
      <label class="label" for="acc-type">Type</label>
      <select id="acc-type" class="field" bind:value={form.type}>
        {#each ACCOUNT_TYPES as type (type)}<option value={type}>{type}</option>{/each}
      </select>
    </div>
    {#if !editing}
      <div>
        <label class="label" for="acc-currency">Entry currency</label>
        <select id="acc-currency" class="field" bind:value={form.currency} aria-describedby="acc-currency-hint">
          <option>VND</option><option>USD</option><option>USDT</option>
        </select>
        <p id="acc-currency-hint" class="muted mt-1 text-xs">
          USD and USDT amounts are converted to VND by the sheet; balances are always VND.
        </p>
      </div>
      <div class="sm:col-span-2">
        <label class="label" for="acc-balance">Opening balance (VND)</label>
        <input id="acc-balance" type="number" step="1" class="field" bind:value={form.balance} />
      </div>
    {/if}
    <div class="sm:col-span-2">
      <label class="label" for="acc-note">Note</label>
      <input id="acc-note" class="field" bind:value={form.note} placeholder="For credit cards: available credit" />
    </div>
    {#if editing}
      <!-- WM-ACC-7: balances change through transactions only -->
      <p class="muted text-xs sm:col-span-2">
        Balances come from transactions in the sheet, so they are not edited here. To correct a balance, add an
        adjustment transaction.
        <a class="link" href="{base}{adjustmentHref(editing.name)}" onclick={onclose}>Add adjustment transaction</a>
      </p>
    {/if}
  </form>
  {#snippet footer()}
    <Button disabled={busy} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="account-form" variant="primary" {busy} busyText="Saving…"
      >{editing ? 'Save' : 'Create'}</Button
    >
  {/snippet}
</Modal>
