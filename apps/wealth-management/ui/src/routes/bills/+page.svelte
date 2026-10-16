<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { BillItem, BillOccurrence, RecurringSuggestion } from '$lib/api/types';
  import {
    billStatus,
    confidenceStatus,
    dueLabel,
    emptyRecurringForm,
    groupItems,
    KIND_LABELS,
    recurringFormFor,
    recurringFormFromSuggestion,
    recurringInputFrom,
    safeToSpendTone,
    scheduleLabel,
    splitCalendar,
    suggestionEvidence,
  } from '$lib/bills/bills';
  import IdleCashCard from '$lib/bills/IdleCashCard.svelte';
  import RecurringFormModal from '$lib/bills/RecurringFormModal.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import { formatDate } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { STATUS } from '$lib/status';

  let { data } = $props();
  const bills = $derived(data.bills);
  const safe = $derived(bills.safeToSpend);
  const calendar = $derived(bills.calendar);
  const sections = $derived(splitCalendar(calendar.occurrences, bills.asOf));
  const groups = $derived(groupItems(bills.items));
  const headlineTone = $derived(safeToSpendTone(safe));

  let busy = $state(false);
  let creating = $state(false);
  let editing = $state<BillItem | null>(null);
  let fromSuggestion = $state<RecurringSuggestion | null>(null);
  let deleting = $state<BillItem | null>(null);
  let form = $state(emptyRecurringForm());

  const formOpen = $derived(creating || editing !== null || fromSuggestion !== null);
  const formTitle = $derived(
    editing ? `Edit ${editing.name}` : fromSuggestion ? `Confirm ${fromSuggestion.name}` : 'New recurring item',
  );

  function closeForm() {
    creating = false;
    editing = null;
    fromSuggestion = null;
  }

  async function run(action: () => Promise<unknown>, success: string, context: string) {
    busy = true;
    try {
      await action();
      toasts.push('success', success);
      closeForm();
      deleting = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, context);
    } finally {
      busy = false;
    }
  }

  function openCreate() {
    form = emptyRecurringForm();
    creating = true;
  }

  function openEdit(item: BillItem) {
    form = recurringFormFor(item);
    editing = item;
  }

  function openSuggestion(suggestion: RecurringSuggestion) {
    form = recurringFormFromSuggestion(suggestion);
    fromSuggestion = suggestion;
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const { input, error } = recurringInputFrom(form);
    if (!input) {
      toasts.push('warning', error ?? 'Check the item.');
      return;
    }
    const current = editing;
    if (current) {
      void run(() => api.updateRecurring(current.id, current.etag, input), `${input.name} saved.`, 'Could not save');
    } else {
      // An edited suggestion is saved as a new item; it then covers the suggestion.
      void run(() => api.createRecurring(input), `${input.name} added.`, 'Could not add the item');
    }
  }

  function confirm(suggestion: RecurringSuggestion) {
    void run(
      () => api.confirmSuggestion({ key: suggestion.key }),
      `${suggestion.name} added.`,
      'Could not confirm the suggestion',
    );
  }

  // Dismissing keeps a paused row, so the suggestion stays dismissed on every device.
  function dismiss(suggestion: RecurringSuggestion) {
    void run(
      () => api.confirmSuggestion({ key: suggestion.key, status: 'paused' }),
      `${suggestion.name} dismissed.`,
      'Could not dismiss the suggestion',
    );
  }

  function toggle(item: BillItem) {
    const next = item.status === 'active' ? 'paused' : 'active';
    void run(
      () => api.setRecurringStatus(item.id, item.etag, next),
      `${item.name} ${next === 'paused' ? 'paused' : 'resumed'}.`,
      'Could not update the item',
    );
  }
</script>

{#snippet occurrenceRow(occurrence: BillOccurrence)}
  {@const status = billStatus(occurrence)}
  <li class="flex flex-wrap items-center justify-between gap-2 py-2.5">
    <div class="min-w-0">
      <p class="truncate font-medium">{occurrence.name}</p>
      <p class="muted text-xs">
        {formatDate(occurrence.expectedDate)} · {dueLabel(occurrence.daysUntil)} · {KIND_LABELS[occurrence.kind]}
        {#if occurrence.estimate}· estimate, can still change{/if}
        {#if occurrence.paidDate}· {occurrence.kind === 'income' ? 'received' : 'paid'}
          {formatDate(occurrence.paidDate)}{/if}
      </p>
    </div>
    <div class="flex items-center gap-3">
      <span class="text-sm">
        {#if occurrence.kind === 'income'}+{/if}<Money value={occurrence.expectedAmount} />
      </span>
      <Badge status={status.tone}>{status.label}</Badge>
    </div>
  </li>
{/snippet}

<PageHeader title="Bills" subtitle="Recurring bills, income and card payments, and what is safe to spend">
  {#snippet actions()}
    <button type="button" class="btn-primary" onclick={openCreate}><Icon name="plus" size={16} /> New item</button>
  {/snippet}
</PageHeader>

<!-- Headline (ROADMAP P2): how much can I spend until the next income? -->
<PageHeadline question="Safe to spend until {formatDate(safe.until)}" tone={headlineTone} id="safe-headline">
  {#snippet answer()}
    <Money value={safe.safeToSpend} class="text-2xl" />
    <span class="text-sm font-normal">
      {#if safe.shortBy > 0}
        Short by <Money value={safe.shortBy} /> before {safe.untilIncome ? 'the next income' : 'the end of the period'}
      {:else}
        About <Money value={safe.perDay} /> a day for {safe.days}
        {safe.days === 1 ? 'day' : 'days'}
      {/if}
    </span>
  {/snippet}
  <dl class="mt-4 grid max-w-md grid-cols-[1fr_auto] gap-x-6 gap-y-1 text-sm">
    <dt>Spendable cash</dt>
    <dd class="text-right"><Money value={safe.spendableCash} /></dd>
    <dt>Bills due by then</dt>
    <dd class="text-right">− <Money value={safe.billsDue} /></dd>
    <dt>Card payments due by then</dt>
    <dd class="text-right">− <Money value={safe.cardPayments} /></dd>
    <dt>Goal set-aside this month</dt>
    <dd class="text-right">− <Money value={safe.goalSetAside} /></dd>
    <dt class="border-t border-slate-900/10 pt-1 font-semibold dark:border-white/10">Left</dt>
    <dd class="border-t border-slate-900/10 pt-1 text-right font-semibold dark:border-white/10">
      <Money value={safe.amount} />
    </dd>
  </dl>
  <p class="muted mt-3 text-xs">
    {#if safe.untilIncome}Until the next income, {safe.incomeName}, expected {formatDate(safe.until)}.{:else}No income
      item is set, so this covers the next 30 days.{/if}
    Spendable cash is active, rarely used, bank and cash accounts.
    {#if safe.goalPlanUsed}The goal set-aside is the full monthly amount your dated goals need.{:else}No goal plan yet,
      so nothing is set aside for goals.{/if}
  </p>
  {#if safe.cashAccounts.length || safe.counted.length}
    <details class="mt-3 text-sm">
      <summary class="cursor-pointer text-xs font-medium">Show what is counted</summary>
      <div class="mt-2 grid gap-4 sm:grid-cols-2">
        <ul class="space-y-1">
          {#each safe.cashAccounts as account (account.name)}
            <li class="flex justify-between gap-2"><span>{account.name}</span><Money value={account.balance} /></li>
          {/each}
        </ul>
        <ul class="space-y-1">
          {#each safe.counted as occurrence (occurrence.name + occurrence.expectedDate)}
            <li class="flex justify-between gap-2">
              <span>{occurrence.name}, {formatDate(occurrence.expectedDate)}</span><Money
                value={occurrence.remaining}
              />
            </li>
          {:else}
            <li class="muted">No unpaid bills by then.</li>
          {/each}
        </ul>
      </div>
    </details>
  {/if}
</PageHeadline>

<IdleCashCard idle={data.idleCash} />

{#if bills.issues.length}
  <Card title="Rows to fix in the Recurring tab" class="mt-4">
    <ul class="space-y-1 text-sm">
      {#each bills.issues as issue (issue.row + issue.message)}
        <li class="flex items-start gap-2 {STATUS.warning.ink}">
          <Icon name={STATUS.warning.icon} size={14} /> Row {issue.row}: {issue.message}
        </li>
      {/each}
    </ul>
  </Card>
{/if}

<div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
  <Card title="Next 30 days" subtitle="Bills, income and card payments until {formatDate(calendar.to)}">
    <div class="mb-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
      <span>To pay: <Money value={calendar.upcomingOut} /> ({calendar.upcomingCount})</span>
      <span>Coming in: <Money value={calendar.upcomingIn} /></span>
      {#if calendar.overdueCount}
        <span class="flex items-center gap-1 {STATUS.serious.ink}">
          <Icon name={STATUS.serious.icon} size={14} />{calendar.overdueCount} overdue, <Money
            value={calendar.overdueAmount}
          />
        </span>
      {/if}
    </div>
    {#if sections.upcoming.length}
      <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each sections.upcoming as occurrence (occurrence.name + occurrence.expectedDate)}
          {@render occurrenceRow(occurrence)}
        {/each}
      </ul>
    {:else}
      <EmptyState
        title="Nothing due in the next 30 days"
        detail="Confirm a suggestion or add an item to see it here."
      />
    {/if}
    {#if sections.earlier.length}
      <details class="mt-3">
        <summary class="cursor-pointer text-sm font-medium">Earlier this month ({sections.earlier.length})</summary>
        <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each sections.earlier as occurrence (occurrence.name + occurrence.expectedDate)}
            {@render occurrenceRow(occurrence)}
          {/each}
        </ul>
      </details>
    {/if}
  </Card>

  <Card
    title="Suggestions"
    subtitle="Found in your transactions. Dismiss keeps a paused row so it is not suggested again."
  >
    {#if bills.suggestions.length}
      <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each bills.suggestions as suggestion (suggestion.key)}
          {@const confidence = confidenceStatus(suggestion.confidence)}
          <li class="py-3">
            <div class="flex flex-wrap items-start justify-between gap-2">
              <div class="min-w-0">
                <p class="truncate font-medium">{suggestion.name}</p>
                <p class="muted text-xs">
                  {KIND_LABELS[suggestion.kind]} · {scheduleLabel(suggestion)} · {suggestion.category}
                </p>
              </div>
              <Money value={suggestion.amount} class="text-sm" />
            </div>
            <p class="muted mt-1 text-xs">{suggestionEvidence(suggestion)}</p>
            <div class="mt-2 flex flex-wrap items-center gap-2">
              <Badge status={confidence.tone}>{confidence.label}</Badge>
              <span class="flex-1"></span>
              <button
                type="button"
                class="btn-ghost px-2 py-1 text-xs"
                disabled={busy}
                onclick={() => dismiss(suggestion)}>Dismiss</button
              >
              <button
                type="button"
                class="btn-ghost px-2 py-1 text-xs"
                disabled={busy}
                onclick={() => openSuggestion(suggestion)}>Edit</button
              >
              <button
                type="button"
                class="btn-primary px-2 py-1 text-xs"
                disabled={busy}
                onclick={() => confirm(suggestion)}>Confirm</button
              >
            </div>
          </li>
        {/each}
      </ul>
    {:else}
      <EmptyState
        title="No suggestions"
        detail="Payments seen in at least 3 of the last 6 months with a steady amount and day show up here."
      />
    {/if}
  </Card>
</div>

<Card title="Recurring items" subtitle="Saved in the Recurring tab of your sheet" class="mt-4">
  {#if groups.active.length}
    <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
      {#each groups.active as item (item.id)}
        {@render itemRow(item)}
      {/each}
    </ul>
  {:else}
    <EmptyState
      title="No recurring items yet"
      detail={bills.tabExists
        ? 'Add salary, rent, utilities or subscriptions to see them on the calendar.'
        : 'Confirming a suggestion or adding an item creates a Recurring tab in your sheet. Your other tabs are not changed.'}
    />
  {/if}
  {#if groups.paused.length}
    <details class="mt-3">
      <summary class="cursor-pointer text-sm font-medium">Paused and dismissed ({groups.paused.length})</summary>
      <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each groups.paused as item (item.id)}
          {@render itemRow(item)}
        {/each}
      </ul>
    </details>
  {/if}
</Card>

{#snippet itemRow(item: BillItem)}
  <li class="flex flex-wrap items-center justify-between gap-2 py-2.5">
    <div class="min-w-0">
      <p class="flex items-center gap-2 font-medium">
        <span class="truncate">{item.name}</span>
        <Badge>{KIND_LABELS[item.kind]}</Badge>
      </p>
      <p class="muted text-xs">
        {scheduleLabel(item)}
        {#if item.match}· payee contains “{item.match}”{/if}
        {#if item.lastMatch}· last {item.kind === 'income' ? 'received' : 'paid'} {formatDate(item.lastMatch.date)}{/if}
        {#if item.note}· {item.note}{/if}
      </p>
    </div>
    <div class="flex items-center gap-2">
      {#if item.amount !== null}<Money value={item.amount} class="text-sm" />{:else}<span class="muted text-sm"
          >Statement</span
        >{/if}
      {#if item.status === 'active' && item.current}
        {@const status = billStatus(item.current)}
        <Badge status={status.tone}>{status.label}</Badge>
      {:else if item.status === 'paused'}
        <Badge status="neutral">Paused</Badge>
      {/if}
      <button
        type="button"
        class="btn-ghost px-2 py-1"
        aria-label="Edit {item.name}"
        disabled={busy}
        onclick={() => openEdit(item)}><Icon name="pencil" size={14} /></button
      >
      <button type="button" class="btn-ghost px-2 py-1 text-xs" disabled={busy} onclick={() => toggle(item)}
        >{item.status === 'active' ? 'Pause' : 'Resume'}</button
      >
      <button
        type="button"
        class="btn-ghost px-2 py-1"
        aria-label="Delete {item.name}"
        disabled={busy}
        onclick={() => (deleting = item)}><Icon name="trash" size={14} /></button
      >
    </div>
  </li>
{/snippet}

<RecurringFormModal
  open={formOpen}
  title={formTitle}
  bind:form
  {busy}
  submitLabel={editing ? 'Save' : 'Add'}
  onclose={closeForm}
  onsubmit={submit}
/>

<Modal open={deleting !== null} title="Delete recurring item" onclose={() => (deleting = null)}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? Its row in the Recurring tab is cleared. To keep it without counting it,
      pause it instead.
    </p>
  {/if}
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (deleting = null)}>Cancel</button>
    <button
      type="button"
      class="btn-danger"
      disabled={busy}
      onclick={() => {
        const item = deleting;
        if (item) void run(() => api.deleteRecurring(item.id, item.etag), `${item.name} deleted.`, 'Could not delete');
      }}>{busy ? 'Deleting…' : 'Delete'}</button
    >
  {/snippet}
</Modal>
