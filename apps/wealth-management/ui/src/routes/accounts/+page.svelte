<script lang="ts">
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { groupAccounts, liquidityBreakdown, sumBalances, totalHistory } from '$lib/accounts';
  import AccountFormModal from '$lib/accounts/AccountFormModal.svelte';
  import AccountGroupSection from '$lib/accounts/AccountGroupSection.svelte';
  import LiquidityCard from '$lib/accounts/LiquidityCard.svelte';
  import { accountFormFor, emptyAccountForm, type AccountForm } from '$lib/accounts/form';
  import { joinStreamed } from '$lib/accounts/streams';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import type { Account } from '$lib/api/types';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import AllocationSummary from '$lib/investments/AllocationSummary.svelte';
  import { action } from '$lib/state/action.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  // The header and its actions render at once; the balance sheet, the
  // allocation, liquidity, the balance history and the account groups stream
  // in, each with its own skeleton and error.
  let { data } = $props();
  const accounts = track(() => data.accounts);
  const histories = track(() => data.histories);
  // The balance history needs both the histories and the accounts.
  const historyData = $derived(joinStreamed(data.histories, data.accounts));

  // Expansion state per group key; defaults from WM-ACC-1.
  let expanded = $state<Record<string, boolean>>({});
  const isExpanded = (key: string, fallback: boolean) => expanded[key] ?? fallback;

  // Create / edit / delete (WM-ACC-6, WM-ACC-7).
  let creating = $state(false);
  let editing = $state<Account | null>(null);
  let deleting = $state<Account | null>(null);
  let form = $state<AccountForm>(emptyAccountForm());

  function openCreate() {
    form = emptyAccountForm();
    creating = true;
  }

  function openEdit(account: Account) {
    form = accountFormFor(account);
    editing = account;
  }

  // A finished change closes the dialogs and reloads the figures; the shown
  // ones stay (with a Refreshing badge) until the new ones arrive.
  async function done(message: string) {
    toasts.success(message);
    creating = false;
    editing = null;
    deleting = null;
    await invalidateAll();
  }

  const create = action(
    async (input: Parameters<typeof api.createAccount>[0]) => {
      await api.createAccount(input);
      await done(`${input.name} created.`);
    },
    { errorContext: 'Could not create the account' },
  );
  const update = action(
    async (name: string, input: Parameters<typeof api.updateAccount>[1]) => {
      await api.updateAccount(name, input);
      await done(`${name} updated.`);
    },
    { errorContext: 'Could not update the account' },
  );
  const remove = action(
    async (name: string) => {
      await api.deleteAccount(name);
      await done(`${name} deleted.`);
    },
    { errorContext: 'Could not delete the account' },
  );
  const saving = $derived(create.busy || update.busy);

  function submit(event: SubmitEvent) {
    event.preventDefault();
    if (editing) {
      void update.run(editing.name, { type: form.type, note: form.note.trim() });
      return;
    }
    const name = form.name.trim();
    if (!name) {
      toasts.push('warning', 'Account name is required.');
      return;
    }
    void create.run({
      name,
      type: form.type,
      currency: form.currency,
      balance: form.balance ?? undefined,
      note: form.note.trim() || undefined,
    });
  }
</script>

<PageHeader
  title="Accounts"
  subtitle={accounts.value
    ? `${accounts.value.length} ${accounts.value.length === 1 ? 'account' : 'accounts'}`
    : undefined}
>
  {#snippet actions()}
    <a class="btn-ghost" href="{base}/accounts/credit-cards"><Icon name="card" size={16} /> Credit cards</a>
    <a class="btn-ghost" href="{base}/accounts/lent"><Icon name="loan" size={16} /> Money lent</a>
    <a class="btn-ghost" href="{base}/accounts/goals"><Icon name="goal" size={16} /> Goals</a>
    <Button variant="primary" icon="plus" onclick={openCreate}>New account</Button>
  {/snippet}
</PageHeader>

<!-- WM-ACC-2 -->
<Deferred data={data.summary} label="the balance sheet">
  {#snippet children(summary)}
    {@const sheet = summary.balanceSheet}
    <div class="grid gap-4 sm:grid-cols-3">
      <Stat label="Total assets" hint={sheet.receivables > 0 ? 'Includes money lent to people' : undefined}
        ><Money value={sheet.assets} /></Stat
      >
      <Stat label="Total liabilities" hint={sheet.payables > 0 ? 'Includes money you owe people' : undefined}
        ><Money value={sheet.liabilities} /></Stat
      >
      <Stat label="Net worth"><Money value={sheet.netWorth} /></Stat>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div class="grid gap-4 sm:grid-cols-3">
      <SkeletonStat label="total assets" />
      <SkeletonStat label="total liabilities" />
      <SkeletonStat label="net worth" />
    </div>
  {/snippet}
</Deferred>

<!-- WM-ACC-8: every figure is VND (D25); the allocation is the engine's one allocation (ROADMAP C4). -->
<div class="mt-4 grid gap-4 lg:grid-cols-3">
  <Deferred data={data.allocation} label="the asset allocation">
    {#snippet children(allocation)}<AllocationSummary {allocation} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Asset allocation" chart={140} lines={0} />{/snippet}
  </Deferred>
  <Deferred data={data.accounts} label="liquidity">
    {#snippet children(list)}<LiquidityCard tiers={liquidityBreakdown(list)} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Liquidity" lines={3} />{/snippet}
  </Deferred>
  <Deferred data={historyData} label="the balance history">
    {#snippet children([trend, list])}
      <Card title="Balance history" subtitle="All accounts, last 12 months">
        <LineChart points={totalHistory(trend, list)} height={180} label="12-month total balance in VND" />
      </Card>
    {/snippet}
    {#snippet skeleton()}<SkeletonCard title="Balance history" chart={180} lines={0} />{/snippet}
  </Deferred>
</div>

<Deferred data={data.accounts} label="accounts" class="mt-4">
  {#snippet children(list)}
    {@const groups = groupAccounts(list)}
    <div>
      {#if groups.length === 0}
        <Card><EmptyState title="No accounts yet" detail="Create one with New account." /></Card>
      {/if}
      {#each groups as { group, accounts: members } (group.key)}
        <AccountGroupSection
          {group}
          accounts={members}
          open={isExpanded(group.key, group.expanded)}
          total={sumBalances(members)}
          histories={histories.value ?? {}}
          ontoggle={() => (expanded[group.key] = !isExpanded(group.key, group.expanded))}
          onedit={openEdit}
          ondelete={(account) => (deleting = account)}
        />
      {/each}
    </div>
  {/snippet}
  {#snippet skeleton()}<SkeletonCard title="Accounts" rows={6} lines={0} />{/snippet}
</Deferred>

<!-- WM-ACC-5 -->
<RecommendationsPanel
  title="Recommendations"
  subtitle="From the account review"
  page="/accounts"
  class="mt-4"
  review={(force, signal) => api.review('account-review', {}, { force, signal })}
/>

<AccountFormModal
  open={creating || editing !== null}
  {editing}
  bind:form
  busy={saving}
  onclose={() => ((creating = false), (editing = null))}
  onsubmit={submit}
/>

<Modal open={deleting !== null} title="Delete account" onclose={() => (deleting = null)} busy={remove.busy}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? The account is soft-deleted in the sheet; its transactions stay.
    </p>
  {/if}
  {#snippet footer()}
    <Button disabled={remove.busy} onclick={() => (deleting = null)}>Cancel</Button>
    <Button
      variant="danger"
      busy={remove.busy}
      busyText="Deleting…"
      onclick={() => {
        if (deleting) void remove.run(deleting.name);
      }}>Delete</Button
    >
  {/snippet}
</Modal>
