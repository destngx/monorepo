<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { track, Button, Card, Deferred, EmptyState, Icon, Modal, SkeletonCard, SkeletonHeadline } from '@libs/ui-kit';
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
    safeToSpendBadge,
    scheduleLabel,
    splitCalendar,
    suggestionEvidence,
  } from '$lib/bills/bills';
  import IdleCashCard from '$lib/bills/IdleCashCard.svelte';
  import MinimumBalances from '$lib/bills/MinimumBalances.svelte';
  import FoldRow from '$lib/components/FoldRow.svelte';
  import SetAside from '$lib/bills/SetAside.svelte';
  import ReadyMoney from '$lib/bills/ReadyMoney.svelte';
  import RecurringFormModal from '$lib/bills/RecurringFormModal.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import { formatDate } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { STATUS } from '$lib/status';

  // The header and New item render at once; the bills report and the idle
  // cash check stream in. A save refreshes them with the old figures shown.
  let { data } = $props();
  const categories = track(() => data.categories);
  // The action running (one at a time), so only its button spins and the others wait.
  let pending = $state<string | null>(null);
  const busy = $derived(pending !== null);
  let formError = $state<string | null>(null);
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

  async function run(key: string, action: () => Promise<unknown>, success: string, context: string) {
    if (pending) return;
    pending = key;
    try {
      await action();
      toasts.push('success', success);
      closeForm();
      deleting = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, context);
    } finally {
      pending = null;
    }
  }

  const isPending = (key: string) => pending === key;
  const waits = (key: string) => pending !== null && pending !== key;

  function openCreate() {
    formError = null;
    form = emptyRecurringForm();
    creating = true;
  }

  function openEdit(item: BillItem) {
    formError = null;
    form = recurringFormFor(item);
    editing = item;
  }

  function openSuggestion(suggestion: RecurringSuggestion) {
    formError = null;
    form = recurringFormFromSuggestion(suggestion);
    fromSuggestion = suggestion;
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const { input, error } = recurringInputFrom(form);
    if (!input) {
      formError = error ?? 'Check the item.';
      return;
    }
    formError = null;
    const current = editing;
    if (current) {
      void run(
        'form',
        () => api.updateRecurring(current.id, current.etag, input),
        `${input.name} saved.`,
        'Could not save',
      );
    } else {
      // An edited suggestion is saved as a new item; it then covers the suggestion.
      void run('form', () => api.createRecurring(input), `${input.name} added.`, 'Could not add the item');
    }
  }

  function confirm(suggestion: RecurringSuggestion) {
    void run(
      `confirm:${suggestion.key}`,
      () => api.confirmSuggestion({ key: suggestion.key }),
      `${suggestion.name} added.`,
      'Could not confirm the suggestion',
    );
  }

  // Dismissing keeps a paused row, so the suggestion stays dismissed on every device.
  function dismiss(suggestion: RecurringSuggestion) {
    void run(
      `dismiss:${suggestion.key}`,
      () => api.confirmSuggestion({ key: suggestion.key, status: 'paused' }),
      `${suggestion.name} dismissed.`,
      'Could not dismiss the suggestion',
    );
  }

  function toggle(item: BillItem) {
    const next = item.status === 'active' ? 'paused' : 'active';
    void run(
      `toggle:${item.id}`,
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
        {#if occurrence.status === 'due' && occurrence.daysUntil < 0 && occurrence.graceUntil}· no rush until
          {formatDate(occurrence.graceUntil)}{/if}
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
    <Button variant="primary" icon="plus" onclick={openCreate}>New item</Button>
  {/snippet}
</PageHeader>

<!-- Headline (ROADMAP P2): how much can I spend until the next income? -->
<Deferred data={data.bills} label="safe to spend">
  {#snippet children(bills)}
    {@const safe = bills.safeToSpend}
    <PageHeadline
      question="Safe to spend until {formatDate(safe.until)}"
      tone={safeToSpendBadge(safe).tone}
      id="safe-headline"
    >
      {#snippet answer()}
        <Money value={safe.safeToSpend} class="text-2xl" />
        <span class="text-sm font-normal">
          {#if safe.shortBy > 0}
            Short by <Money value={safe.shortBy} /> before {safe.untilIncome
              ? 'the next income'
              : 'the end of the period'}
          {:else if safe.tightBy > 0}
            Essentials short by <Money value={safe.tightBy} /> before {safe.untilIncome
              ? 'the next income'
              : 'the end of the period'}
          {:else}
            About <Money value={safe.perDay} /> a day for {safe.days}
            {safe.days === 1 ? 'day' : 'days'}
          {/if}
        </span>
      {/snippet}
      <!-- The answer's four figures; the details fold below, closed until asked for. -->
      <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-4" data-testid="safe-figures">
        <div class="min-w-0">
          <dt class="label">Ready money</dt>
          <dd class="tabular font-semibold"><Money value={safe.readyMoney} /></dd>
        </div>
        <div class="min-w-0">
          <dt class="label">Committed</dt>
          <dd class="tabular font-semibold">
            − <Money value={safe.billsDue + safe.cardPayments + (safe.natureTagged ? safe.fixedCosts : 0)} />
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="label">Essentials</dt>
          <dd class="tabular font-semibold">
            {#if safe.natureTagged}− <Money value={safe.essentials} />{:else}<span class="muted">not set aside</span
              >{/if}
          </dd>
        </div>
        <div class="min-w-0">
          <dt class="label">Free to spend</dt>
          <dd class="tabular font-semibold"><Money value={safe.safeToSpend} /></dd>
        </div>
      </dl>
      {#if !safe.natureTagged}<SetAside {safe} />{/if}
      <div
        class="mt-3 divide-y divide-stone-900/5 border-t border-stone-900/5 text-sm dark:divide-white/5 dark:border-white/5"
      >
        <FoldRow title="How it adds up">
          <dl class="grid max-w-md grid-cols-[1fr_auto] gap-x-6 gap-y-1 text-sm" data-testid="safe-breakdown">
            <dt>
              Ready money{#if safe.reserved > 0}, above minimum balances{/if}
            </dt>
            <dd class="text-right"><Money value={safe.readyMoney} /></dd>
            {#if safe.reserved > 0}
              <dt class="muted text-xs">Kept as minimum balances (not counted)</dt>
              <dd class="muted text-right text-xs" data-testid="safe-reserved"><Money value={safe.reserved} /></dd>
            {/if}
            <dt>Bills due {safe.untilIncome ? 'before payday' : 'by then'}</dt>
            <dd class="text-right">− <Money value={safe.billsDue} /></dd>
            <dt>Card payments due {safe.untilIncome ? 'before payday' : 'by then'}</dt>
            <dd class="text-right">− <Money value={safe.cardPayments} /></dd>
            {#if safe.natureTagged}
              <dt>Fixed costs still to pay</dt>
              <dd class="text-right">− <Money value={safe.fixedCosts} /></dd>
            {/if}
            <dt class="border-t border-stone-900/10 pt-1 dark:border-white/10">Left after committed</dt>
            <dd class="border-t border-stone-900/10 pt-1 text-right dark:border-white/10">
              <Money value={safe.amount} />
            </dd>
            {#if safe.natureTagged}
              <dt>Essentials for {safe.days} {safe.days === 1 ? 'day' : 'days'}</dt>
              <dd class="text-right">− <Money value={safe.essentials} /></dd>
            {/if}
            <dt class="border-t border-stone-900/10 pt-1 font-semibold dark:border-white/10">Free to spend</dt>
            <dd class="border-t border-stone-900/10 pt-1 text-right font-semibold dark:border-white/10">
              <Money value={safe.safeToSpend} />
            </dd>
          </dl>
          <p class="muted mt-3 text-xs" data-testid="safe-explanation">
            {#if safe.untilIncome}Until the next income, {safe.incomeName}, expected {formatDate(safe.until)}. Unpaid
              bills and card payments due before that day count, overdue ones too; anything due on payday itself is paid
              by that wage, so it is not counted.{:else}No income item is set, so this covers the next 30 days: unpaid
              bills and card payments due up to and including {formatDate(safe.until)}, overdue ones too.{/if}
            Ready money is bank, cash, active and rarely used accounts you can spend today; crypto and accounts you mark not
            ready to use (slow to withdraw) are left out. An account with a minimum balance to keep counts only what is above
            that minimum: it is a hard reserve, since a reward on the month's average cannot be saved by topping up late.
            {#if safe.natureTagged}Committed money is never spendable: bills, card payments and the fixed costs (rent,
              insurance) still to pay this month, from their budget or their usual amount. Essentials (groceries,
              transport) are kept for needs at your usual daily pace. Only what is free to spend is for wants.{/if}
          </p>
        </FoldRow>
        {#if safe.natureTagged}
          <FoldRow title="Set aside" money={safe.fixedCosts + safe.essentials}><SetAside {safe} /></FoldRow>
        {/if}
        <FoldRow title="Accounts and what is due" count={safe.counted.length}><ReadyMoney {safe} /></FoldRow>
        <FoldRow title="Minimum balances" money={safe.reserved > 0 ? safe.reserved : undefined}
          ><MinimumBalances {safe} /></FoldRow
        >
      </div>
    </PageHeadline>
  {/snippet}
  {#snippet skeleton()}<SkeletonHeadline question="Safe to spend" label="safe to spend" />{/snippet}
</Deferred>

<Deferred data={data.idleCash} label="idle cash">
  {#snippet children(idle)}<IdleCashCard {idle} />{/snippet}
  {#snippet skeleton()}<SkeletonCard title="Idle cash" lines={3} class="mt-4" />{/snippet}
</Deferred>

<Deferred data={data.bills} label="bills and recurring items">
  {#snippet children(bills)}
    {@const sections = splitCalendar(bills.calendar.occurrences, bills.asOf)}
    {@const groups = groupItems(bills.items)}
    <div>
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

      <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <Card title="Next 30 days" subtitle="Bills, income and card payments until {formatDate(bills.calendar.to)}">
          <div class="mb-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
            <span>To pay: <Money value={bills.calendar.upcomingOut} /> ({bills.calendar.upcomingCount})</span>
            <span>Coming in: <Money value={bills.calendar.upcomingIn} /></span>
            {#if bills.calendar.overdueCount}
              <span class="flex items-center gap-1 {STATUS.serious.ink}">
                <Icon name={STATUS.serious.icon} size={14} />{bills.calendar.overdueCount} overdue, <Money
                  value={bills.calendar.overdueAmount}
                />
              </span>
            {/if}
          </div>
          {#if sections.upcoming.length}
            <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
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
              <summary class="link w-fit text-sm font-medium">Earlier this month ({sections.earlier.length})</summary>
              <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
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
            <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
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
                    <Button
                      size="sm"
                      busy={isPending(`dismiss:${suggestion.key}`)}
                      busyText="Dismissing…"
                      disabled={waits(`dismiss:${suggestion.key}`)}
                      onclick={() => dismiss(suggestion)}>Dismiss</Button
                    >
                    <Button size="sm" disabled={busy} onclick={() => openSuggestion(suggestion)}>Edit</Button>
                    <Button
                      size="sm"
                      variant="primary"
                      busy={isPending(`confirm:${suggestion.key}`)}
                      busyText="Confirming…"
                      disabled={waits(`confirm:${suggestion.key}`)}
                      onclick={() => confirm(suggestion)}>Confirm</Button
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
          <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
            {#each groups.active as item (item.id)}
              {@render itemRow(item)}
            {/each}
          </ul>
        {:else}
          <EmptyState
            title="No recurring items yet"
            detail={bills.tabExists
              ? 'Add salary, rent, utilities or subscriptions to see them on the bills.calendar.'
              : 'Confirming a suggestion or adding an item creates a Recurring tab in your sheet. Your other tabs are not changed.'}
          />
        {/if}
        {#if groups.paused.length}
          <details class="mt-3">
            <summary class="link w-fit text-sm font-medium">Paused and dismissed ({groups.paused.length})</summary>
            <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
              {#each groups.paused as item (item.id)}
                {@render itemRow(item)}
              {/each}
            </ul>
          </details>
        {/if}
      </Card>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
      <SkeletonCard title="Next 30 days" lines={1} rows={5} />
      <SkeletonCard title="Suggestions" lines={4} />
    </div>
    <SkeletonCard title="Recurring items" rows={4} lines={0} class="mt-4" />
  {/snippet}
</Deferred>

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
      <Button size="icon" icon="pencil" label="Edit {item.name}" disabled={busy} onclick={() => openEdit(item)} />
      <Button
        size="sm"
        busy={isPending(`toggle:${item.id}`)}
        disabled={waits(`toggle:${item.id}`)}
        onclick={() => toggle(item)}>{item.status === 'active' ? 'Pause' : 'Resume'}</Button
      >
      <Button size="icon" icon="trash" label="Delete {item.name}" disabled={busy} onclick={() => (deleting = item)} />
    </div>
  </li>
{/snippet}

<RecurringFormModal
  open={formOpen}
  title={formTitle}
  bind:form
  categories={categories.value ?? []}
  categoriesLoading={categories.loading}
  categoriesError={categories.failure?.message ?? null}
  busy={isPending('form')}
  error={formError}
  submitLabel={editing ? 'Save' : 'Add'}
  onclose={closeForm}
  onsubmit={submit}
/>

<Modal
  open={deleting !== null}
  title="Delete recurring item"
  onclose={() => (deleting = null)}
  busy={isPending('delete')}
>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? Its row in the Recurring tab is cleared. To keep it without counting it,
      pause it instead.
    </p>
  {/if}
  {#snippet footer()}
    <Button disabled={isPending('delete')} onclick={() => (deleting = null)}>Cancel</Button>
    <Button
      variant="danger"
      busy={isPending('delete')}
      busyText="Deleting…"
      onclick={() => {
        const item = deleting;
        if (item)
          void run(
            'delete',
            () => api.deleteRecurring(item.id, item.etag),
            `${item.name} deleted.`,
            'Could not delete',
          );
      }}>Delete</Button
    >
  {/snippet}
</Modal>
