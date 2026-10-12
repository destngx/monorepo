<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import BudgetAdvisor from '$lib/ai/BudgetAdvisor.svelte';
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { api } from '$lib/api/endpoints';
  import type { BudgetLine } from '$lib/api/types';
  import BudgetOverview from '$lib/budget/BudgetOverview.svelte';
  import BudgetWall from '$lib/budget/BudgetWall.svelte';
  import CategoryDetailModal from '$lib/budget/CategoryDetailModal.svelte';
  import ExpenseCategoriesCard from '$lib/budget/ExpenseCategoriesCard.svelte';
  import LimitModal from '$lib/budget/LimitModal.svelte';
  import ReallocateModal from '$lib/budget/ReallocateModal.svelte';
  import type { ReallocationDraft } from '$lib/budget/reallocate';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { currentMonth, formatMonth, shiftMonth } from '$lib/format';

  let { data } = $props();
  const report = $derived(data.report);

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
  // Limit editing (WM-BUD-6).
  let editing = $state<BudgetLine | null>(null);
  let limitScope = $state<'month' | 'year'>('month');
  let limitValue = $state<number | null>(null);
  function openEditor(line: BudgetLine) {
    editing = line;
    limitScope = 'month';
    limitValue = line.limit;
  }

  // Move limit between categories, total unchanged (WM-BUD-7).
  let reallocating = $state(false);
  let draft = $state<ReallocationDraft>({ from: '', to: '', amount: null });
  let draftTouched = $state(false);
  // Only a category with a limit can give; any expense category can receive.
  const givers = $derived(expenseLines.filter((line) => line.limit > 0));

  function openReallocate(from = '') {
    draft = { from, to: '', amount: null };
    draftTouched = false;
    reallocating = true;
  }
</script>

<PageHeader title="Budget" subtitle={formatMonth(data.month)}>
  {#snippet actions()}
    <button
      type="button"
      class="btn-ghost"
      disabled={!givers.length || expenseLines.length < 2}
      onclick={() => openReallocate()}>Move money</button
    >
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
<BudgetOverview {report} />

<ExpenseCategoriesCard lines={expenseLines} onselect={(line) => (selected = line)} onedit={openEditor} />

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

<!-- WM-BUD-5 -->
<div class="mt-4 grid gap-4 xl:grid-cols-2">
  <InsightCard
    title="AI budget review"
    subtitle={formatMonth(data.month)}
    load={(force, signal) => api.review('budget-review', { month: data.month }, { force, signal })}
  />
  <BudgetAdvisor />
</div>

<BudgetWall month={data.month} />

<CategoryDetailModal
  line={selected}
  month={data.month}
  transactions={data.transactions}
  onclose={() => (selected = null)}
  onmove={(from) => {
    selected = null;
    openReallocate(from);
  }}
/>

<LimitModal
  line={editing}
  month={data.month}
  bind:scope={limitScope}
  bind:value={limitValue}
  onclose={() => (editing = null)}
/>

<ReallocateModal
  open={reallocating}
  month={data.month}
  {expenseLines}
  {givers}
  bind:draft
  bind:touched={draftTouched}
  onclose={() => (reallocating = false)}
/>
