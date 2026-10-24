<script lang="ts">
  import { openOnPendingFill } from '$lib/fill/pending.svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { track, Button, Card, Deferred, SkeletonCard, SkeletonHeadline } from '@libs/ui-kit';
  import BudgetAdvisor from '$lib/ai/BudgetAdvisor.svelte';
  import type { BudgetLine } from '$lib/api/types';
  import BudgetOverview from '$lib/budget/BudgetOverview.svelte';
  import BudgetWall from '$lib/budget/BudgetWall.svelte';
  import CategoryDetailModal from '$lib/budget/CategoryDetailModal.svelte';
  import CategoryNatureCard from '$lib/budget/CategoryNatureCard.svelte';
  import ExpenseCategoriesCard from '$lib/budget/ExpenseCategoriesCard.svelte';
  import LimitModal from '$lib/budget/LimitModal.svelte';
  import LimitsFillModal from '$lib/budget/LimitsFillModal.svelte';
  import ReallocateModal from '$lib/budget/ReallocateModal.svelte';
  import SpendingRhythmCard from '$lib/budget/SpendingRhythmCard.svelte';
  import type { ReallocationDraft } from '$lib/budget/reallocate';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { currentMonth, formatMonth, shiftMonth } from '$lib/format';

  // The header and month switcher render at once; the report, transactions
  // and categories stream in, each section with its own skeleton and error.
  let { data } = $props();
  const report = track(() => data.report, { scope: () => data.month });
  const transactions = track(() => data.transactions);
  const spending = track(() => data.spending);

  // Lending categories are neither income nor expense (WM-RULE-3).
  const isExpense = (line: BudgetLine) => line.categoryType === 'expense' && !line.lending;
  const lines = $derived(report.value?.lines ?? []);
  const expenseLines = $derived(lines.filter(isExpense).sort((a, b) => b.percentUsed - a.percentUsed));
  const otherLines = $derived(lines.filter((line) => !isExpense(line)));

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

  // Several limits from one sentence (WM-FILL-5).
  let settingLimits = $state(false);

  function openReallocate(from = '') {
    draft = { from, to: '', amount: null };
    draftTouched = false;
    reallocating = true;
  }

  // A chat fill opens its form here (WM-FILL-7).
  openOnPendingFill('budget-limits', () => (settingLimits = true));
  openOnPendingFill('budget-move', () => openReallocate());
</script>

<PageHeader title="Budget" subtitle={formatMonth(data.month)}>
  {#snippet actions()}
    <Button icon="sparkles" disabled={!expenseLines.length} onclick={() => (settingLimits = true)}>Set limits</Button>
    <Button disabled={!givers.length || expenseLines.length < 2} onclick={() => openReallocate()}>Move money</Button>
    <div class="flex items-center gap-1">
      <Button
        size="icon"
        icon="chevronLeft"
        label="Previous month"
        onclick={() => goMonth(shiftMonth(data.month, -1))}
      />
      <Button disabled={data.month === currentMonth()} onclick={() => goMonth(currentMonth())}>Today</Button>
      <Button size="icon" icon="chevronRight" label="Next month" onclick={() => goMonth(shiftMonth(data.month, 1))} />
    </div>
  {/snippet}
</PageHeader>

<!-- WM-BUD-3 overview, then the categories: one report, streamed in for the month -->
<Deferred data={data.report} scope={data.month} label="the budget for {formatMonth(data.month)}">
  {#snippet children(value)}
    <div>
      <BudgetOverview report={value} />
      <ExpenseCategoriesCard lines={expenseLines} onselect={(line) => (selected = line)} onedit={openEditor} />
      {#if otherLines.length}
        <Card title="Income, non-budget and lending" class="mt-4">
          <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5">
            {#each otherLines as line (line.category)}
              <li class="flex justify-between gap-3 py-2">
                <span
                  >{line.category}
                  <span class="muted text-xs">({line.lending ? 'lending' : line.categoryType})</span></span
                >
                <!-- An income line counts money received; the limit is the amount expected. -->
                <span class="tabular"
                  ><Money value={line.spent} /> / <Money
                    value={line.limit}
                  />{#if line.categoryType === 'income' && !line.lending}
                    <span class="muted text-xs">received</span>{/if}</span
                >
              </li>
            {/each}
          </ul>
        </Card>
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonHeadline question="Am I within budget?" label="the budget" stats={4} />
      <SkeletonCard title="Expense categories" rows={6} lines={1} class="mt-4" />
    </div>
  {/snippet}
</Deferred>

<!-- WM-BUD-12: how spending is going today, over 30 days and 3 months, whatever the month shown -->
<Deferred data={data.spending} label="how you are spending" class="mt-4">
  {#snippet children(value)}<SpendingRhythmCard spending={value} />{/snippet}
  {#snippet skeleton()}<SkeletonCard
      title="How am I spending?"
      label="how you are spending"
      lines={3}
      chart={180}
    />{/snippet}
</Deferred>

<!-- WM-BUD-5, WM-BUD-11: the advisor is the page's one AI section, with the actions it is tracking. -->
<BudgetAdvisor class="mt-4" after={Promise.all([data.report, data.transactions, data.categories])} />

<BudgetWall month={data.month} />

<Deferred data={data.categories} label="categories" class="mt-4">
  {#snippet children(categories)}<CategoryNatureCard {categories} />{/snippet}
  {#snippet skeleton()}<SkeletonCard title="Categories" lines={5} />{/snippet}
</Deferred>

<CategoryDetailModal
  line={selected}
  month={data.month}
  transactions={transactions.value ?? null}
  spending={spending.value ?? null}
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

<LimitsFillModal open={settingLimits} month={data.month} lines={expenseLines} onclose={() => (settingLimits = false)} />

<ReallocateModal
  open={reallocating}
  month={data.month}
  {expenseLines}
  {givers}
  bind:draft
  bind:touched={draftTouched}
  onclose={() => (reallocating = false)}
/>
