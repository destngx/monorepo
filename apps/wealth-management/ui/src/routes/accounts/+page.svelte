<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import {
    allocationByGroup,
    fromVND,
    groupAccounts,
    liquidityBreakdown,
    sumInVND,
    totalHistory,
    unconvertible,
    type DisplayCurrency,
  } from '$lib/accounts';
  import AccountFormModal from '$lib/accounts/AccountFormModal.svelte';
  import AccountGroupSection from '$lib/accounts/AccountGroupSection.svelte';
  import AccountOverview from '$lib/accounts/AccountOverview.svelte';
  import { accountFormFor, emptyAccountForm, type AccountForm } from '$lib/accounts/form';
  import { api } from '$lib/api/endpoints';
  import type { Account } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();

  const groups = $derived(groupAccounts(data.accounts));

  // Cross-currency view (WM-ACC-8): every figure below the toggle is converted from VND.
  let display = $state<DisplayCurrency>('VND');
  const rates = $derived(data.fx.rates);
  const inDisplay = (vnd: number) => fromVND(vnd, display, rates);
  const missing = $derived(unconvertible(data.accounts, rates));
  const allocation = $derived(
    allocationByGroup(data.accounts, rates).map((slice) => ({ ...slice, value: inDisplay(slice.value) ?? 0 })),
  );
  const tierRows = $derived(
    liquidityBreakdown(data.accounts, rates).map((tier) => ({ ...tier, shown: inDisplay(tier.value) })),
  );
  const history = $derived(
    totalHistory(data.histories, data.accounts, rates).map((p) => ({ date: p.date, value: inDisplay(p.value) ?? 0 })),
  );
  const usdUnavailable = $derived(display === 'USD' && inDisplay(1) === null);
  // Expansion state per group key; defaults from WM-ACC-1.
  let expanded = $state<Record<string, boolean>>({});
  const isExpanded = (key: string, fallback: boolean) => expanded[key] ?? fallback;

  // Create / edit / delete (WM-ACC-6, WM-ACC-7).
  let creating = $state(false);
  let editing = $state<Account | null>(null);
  let deleting = $state<Account | null>(null);
  let busy = $state(false);
  let form = $state<AccountForm>(emptyAccountForm());

  function openCreate() {
    form = emptyAccountForm();
    creating = true;
  }

  function openEdit(account: Account) {
    form = accountFormFor(account);
    editing = account;
  }

  async function run(action: () => Promise<unknown>, success: string, context: string) {
    busy = true;
    try {
      await action();
      toasts.push('success', success);
      creating = false;
      editing = null;
      deleting = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, context);
    } finally {
      busy = false;
    }
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    if (editing) {
      const name = editing.name;
      void run(
        () => api.updateAccount(name, { type: form.type, note: form.note.trim() }),
        `${name} updated.`,
        'Could not update the account',
      );
    } else {
      if (!form.name.trim()) {
        toasts.push('warning', 'Account name is required.');
        return;
      }
      void run(
        () =>
          api.createAccount({
            name: form.name.trim(),
            type: form.type,
            currency: form.currency,
            balance: form.balance ?? undefined,
            note: form.note.trim() || undefined,
          }),
        `${form.name.trim()} created.`,
        'Could not create the account',
      );
    }
  }
</script>

<PageHeader title="Accounts" subtitle="{data.accounts.length} accounts">
  {#snippet actions()}
    <a class="btn-ghost" href="{base}/accounts/credit-cards"><Icon name="card" size={16} /> Credit cards</a>
    <a class="btn-ghost" href="{base}/accounts/loans"><Icon name="loan" size={16} /> Loans</a>
    <a class="btn-ghost" href="{base}/accounts/goals"><Icon name="goal" size={16} /> Goals</a>
    <button type="button" class="btn-primary" onclick={openCreate}><Icon name="plus" size={16} /> New account</button>
  {/snippet}
</PageHeader>

<div class="mb-3 flex flex-wrap items-center justify-end gap-3">
  {#if data.fx.errors.length}
    <p class="text-xs text-orange-600 dark:text-orange-400" role="status">{data.fx.errors.join('; ')}</p>
  {/if}
  <div class="flex rounded-xl bg-slate-900/5 p-0.5 text-xs dark:bg-white/5" role="group" aria-label="Display currency">
    {#each ['VND', 'USD'] as const as currency (currency)}
      <button
        type="button"
        aria-pressed={display === currency}
        class="rounded-lg px-2.5 py-1 {display === currency ? 'bg-white shadow-sm dark:bg-slate-700' : 'muted'}"
        onclick={() => (display = currency)}>{currency}</button
      >
    {/each}
  </div>
</div>

<!-- WM-ACC-2 -->
<div class="grid gap-4 sm:grid-cols-3">
  <Stat label="Total assets"><Money value={inDisplay(data.balanceSheet.assets)} currency={display} /></Stat>
  <Stat label="Total liabilities"><Money value={inDisplay(data.balanceSheet.liabilities)} currency={display} /></Stat>
  <Stat label="Net worth"><Money value={inDisplay(data.balanceSheet.netWorth)} currency={display} /></Stat>
</div>
{#if usdUnavailable}
  <p class="mt-2 text-xs text-orange-600 dark:text-orange-400">
    USD figures need the USD/VND rate, which did not load.
  </p>
{/if}
{#if missing.length}
  <p class="muted mt-2 text-xs">
    Left out of converted totals (no rate): {missing.map((account) => account.name).join(', ')}
  </p>
{/if}

<!-- WM-ACC-8 -->
<AccountOverview {display} {allocation} tiers={tierRows} {history} />

{#if groups.length === 0}
  <Card class="mt-4"><EmptyState title="No accounts yet" detail="Create one with New account." /></Card>
{/if}

{#each groups as { group, accounts } (group.key)}
  <AccountGroupSection
    {group}
    {accounts}
    open={isExpanded(group.key, group.expanded)}
    total={inDisplay(sumInVND(accounts, rates))}
    currency={display}
    histories={data.histories}
    ontoggle={() => (expanded[group.key] = !isExpanded(group.key, group.expanded))}
    onedit={openEdit}
    ondelete={(account) => (deleting = account)}
  />
{/each}

<!-- WM-ACC-5 -->
<InsightCard
  title="AI account review"
  class="mt-4"
  load={(force, signal) => api.review('account-review', {}, { force, signal })}
/>

<AccountFormModal
  open={creating || editing !== null}
  {editing}
  bind:form
  {busy}
  onclose={() => ((creating = false), (editing = null))}
  onsubmit={submit}
/>

<Modal open={deleting !== null} title="Delete account" onclose={() => (deleting = null)}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? The account is soft-deleted in the sheet; its transactions stay.
    </p>
  {/if}
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (deleting = null)}>Cancel</button>
    <button
      type="button"
      class="btn-danger"
      disabled={busy}
      onclick={() => {
        const name = deleting?.name ?? '';
        void run(() => api.deleteAccount(name), `${name} deleted.`, 'Could not delete the account');
      }}>{busy ? 'Deleting…' : 'Delete'}</button
    >
  {/snippet}
</Modal>
