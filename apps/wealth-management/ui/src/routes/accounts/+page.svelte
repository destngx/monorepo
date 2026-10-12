<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import {
    accountIcon,
    adjustmentHref,
    allocationByGroup,
    creditRemaining,
    creditUtilization,
    fromVND,
    groupAccounts,
    isCreditAccount,
    liquidityBreakdown,
    sumInVND,
    totalHistory,
    unconvertible,
    type DisplayCurrency,
  } from '$lib/accounts';
  import { api } from '$lib/api/endpoints';
  import { ACCOUNT_TYPES, type Account, type AccountType } from '$lib/api/types';
  import Donut from '$lib/charts/Donut.svelte';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';
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
  const tiers = $derived(liquidityBreakdown(data.accounts, rates));
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
  let form = $state({
    name: '',
    type: 'active use' as AccountType,
    currency: 'VND',
    balance: null as number | null,
    note: '',
  });

  function openCreate() {
    form = { name: '', type: 'active use', currency: 'VND', balance: null, note: '' };
    creating = true;
  }

  function openEdit(account: Account) {
    form = {
      name: account.name,
      type: account.type,
      currency: account.currency,
      balance: null,
      note: account.note ?? '',
    };
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
<div class="mt-4 grid gap-4 lg:grid-cols-3">
  <Card title="Allocation by group">
    {#if allocation.length}
      <div class="flex flex-wrap items-center gap-5">
        <Donut items={allocation} size={140} label="Allocation by account group" />
        <ul class="min-w-0 flex-1 space-y-1 text-sm">
          {#each allocation as slice (slice.label)}
            <li class="flex justify-between gap-3">
              <span class="truncate">{slice.label}</span><Money value={slice.value} currency={display} />
            </li>
          {/each}
        </ul>
      </div>
    {:else}
      <p class="muted text-sm">No positive balances to allocate.</p>
    {/if}
  </Card>
  <Card title="Liquidity" subtitle="Immediate: spendable now · T+2: a few days · Illiquid: long holding">
    <ul class="space-y-3">
      {#each tiers as tier (tier.tier)}
        <li>
          <div class="mb-1 flex justify-between gap-3 text-sm">
            <span>{tier.tier}</span>
            <span class="muted tabular whitespace-nowrap"
              ><Money value={inDisplay(tier.value)} currency={display} /> · {formatPercent(tier.percent, 0)}</span
            >
          </div>
          <ProgressBar percent={tier.percent} color="bg-indigo-500" label="{tier.tier} share" />
        </li>
      {/each}
    </ul>
  </Card>
  <Card title="Balance history" subtitle="All accounts, last 12 months at today's rates">
    <LineChart points={history} height={180} label="12-month total balance in {display}" />
  </Card>
</div>

{#if groups.length === 0}
  <Card class="mt-4"><EmptyState title="No accounts yet" detail="Create one with New account." /></Card>
{/if}

{#each groups as { group, accounts } (group.key)}
  {@const open = isExpanded(group.key, group.expanded)}
  <section class="glass mt-4">
    <button
      type="button"
      class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left"
      aria-expanded={open}
      onclick={() => (expanded[group.key] = !open)}
    >
      <span class="flex items-center gap-2 font-semibold">
        <Icon name="chevronDown" size={16} class="transition {open ? '' : '-rotate-90'}" />
        {group.label}
        <span class="muted text-xs font-normal">({accounts.length})</span>
      </span>
      {#if group.subtotal}<Money
          value={inDisplay(sumInVND(accounts, rates))}
          currency={display}
          class="font-semibold"
        />{/if}
    </button>
    {#if open}
      <ul class="divide-y divide-slate-900/5 border-t border-slate-900/5 dark:divide-white/5 dark:border-white/5">
        {#each accounts as account (account.name)}
          {@const utilization = isCreditAccount(account) ? creditUtilization(account) : null}
          {@const recent = (data.histories[account.name] ?? []).slice(-60)}
          <li class="grid items-center gap-3 px-5 py-3 sm:grid-cols-[minmax(0,1fr)_8rem_12rem_auto]">
            <div class="flex min-w-0 items-center gap-3">
              <span class="grid size-9 shrink-0 place-items-center rounded-xl bg-slate-900/5 dark:bg-white/5"
                ><Icon name={accountIcon(account.type)} size={16} /></span
              >
              <div class="min-w-0">
                <p class="truncate font-medium">{account.name}</p>
                <p class="muted truncate text-xs">
                  {account.type}{account.currency !== 'VND' ? ` · ${account.currency}` : ''}{account.dueDate
                    ? ` · target ${formatDate(account.dueDate)}`
                    : ''}{account.note && !isCreditAccount(account) ? ` · ${account.note}` : ''}
                </p>
              </div>
            </div>
            <div class="hidden sm:block">
              <Sparkline
                values={recent.map((p) => p.value)}
                width={128}
                height={32}
                label="{account.name} 60-day balance"
              />
            </div>
            <div class="text-right">
              <Money value={account.balance} currency={account.currency} class="font-medium" />
              {#if account.clearedBalance !== account.balance}
                <p class="muted text-xs">
                  Cleared <Money value={account.clearedBalance} currency={account.currency} />
                </p>
              {/if}
              {#if isCreditAccount(account)}
                <p class="muted text-xs">Available <Money value={creditRemaining(account)} /></p>
              {/if}
              {#if utilization !== null}
                <div class="mt-1 flex items-center gap-2">
                  <ProgressBar
                    percent={utilization}
                    color={utilization > 80 ? 'bg-orange-500' : 'bg-emerald-500'}
                    label="Utilisation"
                  />
                  <span class="muted text-xs">{formatPercent(utilization, 0)}</span>
                </div>
              {:else if account.goalProgress !== null && account.goalAmount}
                <div class="mt-1 flex items-center gap-2">
                  <ProgressBar percent={account.goalProgress} color="bg-indigo-500" label="Goal progress" />
                  <span class="muted text-xs">{formatPercent(account.goalProgress, 0)}</span>
                </div>
              {/if}
            </div>
            <div class="flex justify-end">
              <button
                type="button"
                class="btn-ghost px-2"
                aria-label="Edit {account.name}"
                onclick={() => openEdit(account)}><Icon name="pencil" size={15} /></button
              >
              <button
                type="button"
                class="btn-ghost px-2"
                aria-label="Delete {account.name}"
                onclick={() => (deleting = account)}><Icon name="trash" size={15} /></button
              >
            </div>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/each}

<!-- WM-ACC-5 -->
<InsightCard
  title="AI account review"
  class="mt-4"
  load={(force, signal) => api.review('account-review', {}, { force, signal })}
/>

<Modal
  open={creating || editing !== null}
  title={editing ? `Edit ${editing.name}` : 'New account'}
  onclose={() => ((creating = false), (editing = null))}
>
  <form id="account-form" class="grid gap-4 sm:grid-cols-2" onsubmit={submit}>
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
        <label class="label" for="acc-currency">Currency</label>
        <select id="acc-currency" class="field" bind:value={form.currency}>
          <option>VND</option><option>USD</option><option>USDT</option>
        </select>
      </div>
      <div class="sm:col-span-2">
        <label class="label" for="acc-balance">Opening balance</label>
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
          onclick={() => (editing = null)}>Add adjustment transaction</a
        >
      </p>
    {/if}
  </form>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => ((creating = false), (editing = null))}>Cancel</button>
    <button type="submit" form="account-form" class="btn-primary" disabled={busy}
      >{busy ? 'Saving…' : editing ? 'Save' : 'Create'}</button
    >
  {/snippet}
</Modal>

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
