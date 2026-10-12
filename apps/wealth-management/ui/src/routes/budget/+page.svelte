<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { api } from '$lib/api/endpoints';
  import type { BudgetLine, BudgetStatus } from '$lib/api/types';
  import { categoryTransactions, dailySpend, detailEnd } from '$lib/budget/detail';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Donut from '$lib/charts/Donut.svelte';
  import { EXPENSE } from '$lib/charts/palette';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { currentMonth, formatDate, formatMonth, formatPercent, shiftMonth, todayISO } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();
  const report = $derived(data.report);

  // WM-BUD-3 status colours and labels.
  const statusTone: Record<BudgetStatus, 'green' | 'amber' | 'red'> = {
    healthy: 'green',
    warning: 'amber',
    urgent: 'red',
  };
  const statusLabel: Record<BudgetStatus, string> = { healthy: 'Healthy', warning: 'Warning', urgent: 'Urgent' };
  const statusBar: Record<BudgetStatus, string> = {
    healthy: 'bg-emerald-500',
    warning: 'bg-amber-400',
    urgent: 'bg-rose-500',
  };

  const expenseLines = $derived(
    report.lines.filter((line) => line.categoryType === 'expense').sort((a, b) => b.percentUsed - a.percentUsed),
  );
  const otherLines = $derived(report.lines.filter((line) => line.categoryType !== 'expense'));

  function goMonth(month: string) {
    const url = new URL(page.url);
    if (month === currentMonth()) url.searchParams.delete('month');
    else url.searchParams.set('month', month);
    void goto(url, { keepFocus: true, noScroll: true });
  }

  // Category detail (WM-BUD-4).
  let selected = $state<BudgetLine | null>(null);
  const detailRows = $derived(selected ? categoryTransactions(data.transactions, selected.category, data.month) : []);
  const detailDays = $derived(
    selected ? dailySpend(data.transactions, selected.category, detailEnd(data.month, todayISO())) : [],
  );

  // Limit editing (WM-BUD-6).
  let editing = $state<BudgetLine | null>(null);
  let limitScope = $state<'month' | 'year'>('month');
  let limitValue = $state<number | null>(null);
  let saving = $state(false);

  function openEditor(line: BudgetLine) {
    editing = line;
    limitScope = 'month';
    limitValue = line.limit;
  }

  async function saveLimit(event: SubmitEvent) {
    event.preventDefault();
    if (!editing || limitValue === null || limitValue < 0) return;
    saving = true;
    try {
      await api.updateBudgetLimit({
        year: Number(data.month.slice(0, 4)),
        category: editing.category,
        // An empty month writes the yearly limit column.
        month: limitScope === 'month' ? data.month : undefined,
        limit: limitValue,
      });
      toasts.push('success', `Limit for ${editing.category} updated.`);
      editing = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not update the limit');
    } finally {
      saving = false;
    }
  }
</script>

<PageHeader title="Budget" subtitle={formatMonth(data.month)}>
  {#snippet actions()}
    <div class="flex items-center gap-1">
      <button
        type="button"
        class="btn-ghost px-2"
        aria-label="Previous month"
        onclick={() => goMonth(shiftMonth(data.month, -1))}><Icon name="chevronLeft" /></button
      >
      <button
        type="button"
        class="btn-ghost"
        disabled={data.month === currentMonth()}
        onclick={() => goMonth(currentMonth())}>Today</button
      >
      <button
        type="button"
        class="btn-ghost px-2"
        aria-label="Next month"
        onclick={() => goMonth(shiftMonth(data.month, 1))}><Icon name="chevronRight" /></button
      >
    </div>
  {/snippet}
</PageHeader>

<!-- WM-BUD-3 overview -->
<Card>
  <div class="flex flex-wrap items-end justify-between gap-4">
    <div>
      <p class="muted text-xs tracking-wide uppercase">Spent of expense budget</p>
      <p class="mt-1 text-2xl font-semibold">
        <Money value={report.totalSpent} />
        <span class="muted text-base font-normal">/ <Money value={report.totalLimit} /></span>
      </p>
    </div>
    <div class="flex items-center gap-2">
      {#if report.percentUsed > 100}<Badge tone="red">Budget Burst</Badge>{/if}
      <Badge tone={statusTone[report.status]}
        >{statusLabel[report.status]} · {formatPercent(report.percentUsed, 0)}</Badge
      >
    </div>
  </div>
  <div class="mt-4">
    <ProgressBar percent={report.percentUsed} color={statusBar[report.status]} label="Budget used" />
  </div>
  <div class="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-4">
    <Stat label="Remaining" value={report.totalRemaining} />
    <Stat label="Daily burn" value={report.dailyBurn} />
    <Stat label="Projected spend" value={report.projectedSpend} />
    <Stat label="Days remaining">
      <span class="tabular">{report.daysRemaining}</span>
      <span class="muted text-sm font-normal">of {report.daysInMonth}</span>
    </Stat>
  </div>
</Card>

<Card title="Expense categories" subtitle="Click a category for detail" class="mt-4">
  {#if expenseLines.length}
    <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
      {#each expenseLines as line (line.category)}
        <li class="flex items-center gap-3 py-3">
          <button type="button" class="min-w-0 flex-1 text-left" onclick={() => (selected = line)}>
            <div class="mb-1.5 flex flex-wrap items-center justify-between gap-2 text-sm">
              <span class="flex items-center gap-2 font-medium">
                {line.category}
                {#if line.unbudgeted}<Badge tone="violet">No limit</Badge>{/if}
              </span>
              <span class="muted tabular text-xs"
                ><Money value={line.spent} /> / <Money value={line.limit} /> · <Money value={line.remaining} /> left</span
              >
            </div>
            <ProgressBar
              percent={line.unbudgeted ? 100 : line.percentUsed}
              color={statusBar[line.status]}
              label={line.category}
            />
          </button>
          <button
            type="button"
            class="btn-ghost px-2"
            aria-label="Edit limit for {line.category}"
            onclick={() => openEditor(line)}><Icon name="pencil" size={15} /></button
          >
        </li>
      {/each}
    </ul>
  {:else}
    <EmptyState
      title="No expense budget for this month"
      detail="Set limits in the Budget sheet or edit a category limit here."
    />
  {/if}
</Card>

{#if otherLines.length}
  <Card title="Income and non-budget" class="mt-4">
    <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
      {#each otherLines as line (line.category)}
        <li class="flex justify-between gap-3 py-2">
          <span>{line.category} <span class="muted text-xs">({line.categoryType})</span></span>
          <span class="tabular"><Money value={line.spent} /> / <Money value={line.limit} /></span>
        </li>
      {/each}
    </ul>
  </Card>
{/if}

<Modal open={selected !== null} title={selected?.category ?? ''} onclose={() => (selected = null)} wide>
  {#if selected}
    <div class="grid items-center gap-6 sm:grid-cols-[auto_1fr]">
      <Donut
        items={[
          { label: 'Spent', value: selected.spent, color: EXPENSE },
          { label: 'Remaining', value: selected.remaining, color: '#10b981' },
        ]}
        size={160}
        label="Spent vs remaining"
      >
        {#snippet center()}
          <span class="text-lg font-semibold">{formatPercent(selected?.percentUsed, 0)}</span>
          <span class="muted text-xs">used</span>
        {/snippet}
      </Donut>
      <dl class="grid grid-cols-2 gap-3 text-sm">
        <div>
          <dt class="muted text-xs">Spent</dt>
          <dd><Money value={selected.spent} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Limit</dt>
          <dd><Money value={selected.limit} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Year spent</dt>
          <dd><Money value={selected.yearlySpent} /></dd>
        </div>
        <div>
          <dt class="muted text-xs">Year remaining</dt>
          <dd><Money value={selected.yearlyRemaining} /></dd>
        </div>
        {#if selected.note}<div class="col-span-2">
            <dt class="muted text-xs">Note</dt>
            <dd>{selected.note}</dd>
          </div>{/if}
      </dl>
    </div>
    <h3 class="mt-6 mb-2 text-sm font-semibold">Daily spend, last 30 days</h3>
    <BarChart
      labels={detailDays.map((d, i) => (i % 5 === 0 ? d.date.slice(8, 10) : ''))}
      series={[{ name: 'Spend', color: EXPENSE, values: detailDays.map((d) => d.value) }]}
      height={160}
      label="Daily spend"
    />
    <h3 class="mt-6 mb-2 text-sm font-semibold">Transactions in {formatMonth(data.month)}</h3>
    {#if detailRows.length}
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each detailRows as tx (tx.id)}
          <li class="flex justify-between gap-3 py-2">
            <span><span class="muted tabular mr-2">{formatDate(tx.date)}</span>{tx.payee}</span>
            <Money value={(tx.deposit ?? 0) - (tx.payment ?? 0)} tone />
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">No transactions this month.</p>
    {/if}
  {/if}
</Modal>

<Modal open={editing !== null} title="Limit for {editing?.category ?? ''}" onclose={() => (editing = null)}>
  <form id="limit-form" class="space-y-4" onsubmit={saveLimit}>
    <fieldset class="space-y-2 text-sm">
      <legend class="label">Apply to</legend>
      <label class="flex items-center gap-2"
        ><input
          type="radio"
          bind:group={limitScope}
          value="month"
          class="accent-indigo-600"
          onchange={() => (limitValue = editing?.limit ?? 0)}
        />
        Monthly limit for {formatMonth(data.month)}</label
      >
      <label class="flex items-center gap-2"
        ><input
          type="radio"
          bind:group={limitScope}
          value="year"
          class="accent-indigo-600"
          onchange={() => (limitValue = editing?.yearlyLimit ?? 0)}
        />
        Yearly limit for {data.month.slice(0, 4)}</label
      >
    </fieldset>
    <div>
      <label class="label" for="limit-value">Limit (VND)</label>
      <input id="limit-value" type="number" min="0" step="1000" class="field" bind:value={limitValue} required />
    </div>
  </form>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (editing = null)}>Cancel</button>
    <button type="submit" form="limit-form" class="btn-primary" disabled={saving}
      >{saving ? 'Saving…' : 'Save limit'}</button
    >
  {/snippet}
</Modal>
