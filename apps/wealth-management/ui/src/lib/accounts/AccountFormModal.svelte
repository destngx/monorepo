<script lang="ts">
  import { base } from '$app/paths';
  import { adjustmentHref } from '$lib/accounts';
  import { ACCOUNT_TYPES, type Account } from '$lib/api/types';
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
</script>

<Modal {open} title={editing ? `Edit ${editing.name}` : 'New account'} {onclose}>
  <form id="account-form" class="grid gap-4 sm:grid-cols-2" {onsubmit}>
    {#if !editing}
      <div class="sm:col-span-2">
        <label class="label" for="acc-name">Name *</label>
        <input id="acc-name" class="field" bind:value={form.name} required />
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
        <a
          class="text-indigo-600 hover:underline dark:text-indigo-400"
          href="{base}{adjustmentHref(editing.name)}"
          onclick={onclose}>Add adjustment transaction</a
        >
      </p>
    {/if}
  </form>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={onclose}>Cancel</button>
    <button type="submit" form="account-form" class="btn-primary" disabled={busy}
      >{busy ? 'Saving…' : editing ? 'Save' : 'Create'}</button
    >
  {/snippet}
</Modal>
