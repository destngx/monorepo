<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { allocationByGroup, groupAccounts, liquidityBreakdown, sumBalances, totalHistory } from '$lib/accounts';
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

  // WM-ACC-8: balances are VND, USD/USDT entries included (D25).
  const allocation = $derived(allocationByGroup(data.accounts));
  const tierRows = $derived(liquidityBreakdown(data.accounts));
  const history = $derived(totalHistory(data.histories, data.accounts));
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
    <a class="btn-ghost" href="{base}/accounts/lent"><Icon name="loan" size={16} /> Money lent</a>
    <a class="btn-ghost" href="{base}/accounts/goals"><Icon name="goal" size={16} /> Goals</a>
    <button type="button" class="btn-primary" onclick={openCreate}><Icon name="plus" size={16} /> New account</button>
  {/snippet}
</PageHeader>

<!-- WM-ACC-2 -->
<div class="grid gap-4 sm:grid-cols-3">
  <Stat label="Total assets" hint={data.balanceSheet.receivables > 0 ? 'Includes money lent to people' : undefined}
    ><Money value={data.balanceSheet.assets} /></Stat
  >
  <Stat label="Total liabilities" hint={data.balanceSheet.payables > 0 ? 'Includes money you owe people' : undefined}
    ><Money value={data.balanceSheet.liabilities} /></Stat
  >
  <Stat label="Net worth"><Money value={data.balanceSheet.netWorth} /></Stat>
</div>

<!-- WM-ACC-8 -->
<AccountOverview {allocation} tiers={tierRows} {history} />

{#if groups.length === 0}
  <Card class="mt-4"><EmptyState title="No accounts yet" detail="Create one with New account." /></Card>
{/if}

{#each groups as { group, accounts } (group.key)}
  <AccountGroupSection
    {group}
    {accounts}
    open={isExpanded(group.key, group.expanded)}
    total={sumBalances(accounts)}
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
