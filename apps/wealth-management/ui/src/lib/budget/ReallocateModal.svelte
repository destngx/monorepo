<script lang="ts">
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { BudgetLine } from '$lib/api/types';
  import { Button, Modal } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatMonth } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { reallocationPreview, reallocationProblems, type ReallocationDraft } from './reallocate';

  // Move limit between categories, total unchanged (WM-BUD-7).
  interface Props {
    open: boolean;
    month: string;
    expenseLines: BudgetLine[];
    /** Only a category with a limit can give; any expense category can receive. */
    givers: BudgetLine[];
    draft: ReallocationDraft;
    /** Shows validation problems once the user has tried to submit. */
    touched: boolean;
    onclose: () => void;
  }

  let { open, month, expenseLines, givers, draft = $bindable(), touched = $bindable(), onclose }: Props = $props();
  let moving = $state(false);
  // Fields AI filled from the Describe it box (WM-FILL-6).
  const marks = new FillMarks();
  $effect(() => {
    if (!open) marks.clear();
  });

  const draftProblems = $derived(reallocationProblems(expenseLines, draft));
  const draftPreview = $derived(reallocationPreview(expenseLines, draft));

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    touched = true;
    if (draftProblems.length || draft.amount === null) return;
    moving = true;
    try {
      await api.reallocateBudget({
        year: Number(month.slice(0, 4)),
        month,
        from: draft.from,
        to: draft.to,
        amount: draft.amount,
      });
      toasts.push('success', `Limit moved from ${draft.from} to ${draft.to}.`);
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not move the limit');
    } finally {
      moving = false;
    }
  }
</script>

<Modal {open} title="Move money between categories" {onclose} busy={moving}>
  <FillBox
    form="budget-move"
    class="mb-4"
    current={() => ({ ...draft, month })}
    onfill={(result) => {
      draft = applyFill(draft, result.values);
      marks.take(result);
    }}
  />
  <form id="reallocate-form" class="space-y-4" onsubmit={submit} novalidate>
    <p class="muted text-sm">
      Moves part of one category's {formatMonth(month)} limit to another. The total budget stays the same.
    </p>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <div>
        <label class="label" for="move-from">From<FilledMark show={marks.has('from', draft.from)} /></label>
        <select id="move-from" class="field" bind:value={draft.from}>
          <option value="" disabled>Choose a category</option>
          {#each givers as line (line.category)}
            <option value={line.category}>{line.category}</option>
          {/each}
        </select>
      </div>
      <div>
        <label class="label" for="move-to">To<FilledMark show={marks.has('to', draft.to)} /></label>
        <select id="move-to" class="field" bind:value={draft.to}>
          <option value="" disabled>Choose a category</option>
          {#each expenseLines as line (line.category)}
            <option value={line.category} disabled={line.category === draft.from}>{line.category}</option>
          {/each}
        </select>
      </div>
    </div>
    <div>
      <label class="label" for="move-amount">Amount (VND)<FilledMark show={marks.has('amount', draft.amount)} /></label>
      <input id="move-amount" type="number" min="0" step="1000" class="field" bind:value={draft.amount} />
    </div>
    {#if draftPreview}
      <p class="text-sm">
        {draft.from}: <Money value={draftPreview.from} /> · {draft.to}: <Money value={draftPreview.to} />
      </p>
    {/if}
    {#if touched && draftProblems.length}
      <ul role="alert" class="space-y-0.5 text-sm text-orange-600 dark:text-orange-400">
        {#each draftProblems as problem (problem)}<li>{problem}</li>{/each}
      </ul>
    {/if}
  </form>
  {#snippet footer()}
    <Button disabled={moving} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="reallocate-form" variant="primary" busy={moving} busyText="Moving…">Move</Button>
  {/snippet}
</Modal>
