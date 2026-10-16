<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { GoalProgress } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatMonth } from '$lib/format';
  import GoalCard from '$lib/goals/GoalCard.svelte';
  import GoalFormModal from '$lib/goals/GoalFormModal.svelte';
  import {
    emptyGoalForm,
    goalFormFor,
    goalInputFrom,
    groupGoals,
    looksLikeEmergencyFund,
    moveId,
    verdictStatus,
  } from '$lib/goals/goals';
  import PoolCard from '$lib/goals/PoolCard.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { STATUS } from '$lib/status';

  let { data } = $props();
  const plan = $derived(data.plan);
  const groups = $derived(groupGoals(plan));
  const check = $derived(plan.check);
  const verdict = $derived(verdictStatus(check.verdict));
  const style = $derived(STATUS[verdict.tone]);

  let busy = $state(false);
  let creating = $state(false);
  let editing = $state<GoalProgress | null>(null);
  let deleting = $state<GoalProgress | null>(null);
  let form = $state(emptyGoalForm());

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

  function openCreate() {
    form = emptyGoalForm();
    creating = true;
  }

  function openEdit(goal: GoalProgress) {
    form = goalFormFor(goal);
    editing = goal;
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const current = editing;
    const { input, error } = goalInputFrom(
      form,
      current?.emergencyFund || looksLikeEmergencyFund(form.name, form.note),
    );
    if (!input) {
      toasts.push('warning', error ?? 'Check the goal.');
      return;
    }
    if (current) {
      void run(
        () => api.updateGoal(current.id, current.etag, input),
        `${input.name} saved.`,
        'Could not save the goal',
      );
    } else {
      void run(() => api.createGoal(input), `${input.name} added.`, 'Could not add the goal');
    }
  }

  // Reorders active goals; paused and done goals keep their places after them.
  function move(index: number, delta: -1 | 1) {
    const active = moveId(
      groups.active.map((goal) => goal.id),
      index,
      delta,
    );
    const ids = [...active, ...groups.paused.map((goal) => goal.id), ...groups.done.map((goal) => goal.id)];
    void run(() => api.reorderGoals(ids), 'Priorities updated.', 'Could not reorder the goals');
  }

  function setStatus(goal: GoalProgress, status: GoalProgress['status'], success: string) {
    const { input } = goalInputFrom({ ...goalFormFor(goal), status }, goal.emergencyFund);
    if (input) void run(() => api.updateGoal(goal.id, goal.etag, input), success, 'Could not update the goal');
  }
</script>

<PageHeader title="Goals" subtitle="Goals share one savings pool, filled in priority order">
  {#snippet actions()}
    {#if !plan.setupNeeded}
      <button type="button" class="btn-primary" onclick={openCreate}><Icon name="plus" size={16} /> New goal</button>
    {/if}
  {/snippet}
</PageHeader>

{#if plan.setupNeeded}
  <Card>
    <EmptyState
      title="Set up your goal plan"
      detail="This adds a Goal Plan tab to your sheet with an emergency fund first{plan.legacyGoals
        ? ` and the ${plan.legacyGoals} goals from your Goals tab`
        : ''}. Your existing tabs are not changed."
    />
    <p class="text-center">
      <button
        type="button"
        class="btn-primary"
        disabled={busy}
        onclick={() => void run(() => api.setupGoalPlan(), 'Goal plan created.', 'Could not set up the goal plan')}
        >{busy ? 'Setting up…' : 'Set up your goal plan'}</button
      >
    </p>
  </Card>
  <div class="mt-4 max-w-xl"><PoolCard pool={plan.pool} asOf={plan.asOf} /></div>
{:else}
  <!-- Headline: am I on track? -->
  <section class="glass border-l-4 p-5 {style.edge}" aria-labelledby="goals-verdict">
    <p class="muted text-xs font-medium uppercase">Am I on track?</p>
    <h2 id="goals-verdict" class="mt-1 text-lg font-semibold">
      {#if check.verdict === 'no_deadlines'}
        You save <Money value={check.monthlySavings} /> a month
      {:else}
        Your goals need <Money value={check.requiredPerMonth} /> a month, you save
        <Money value={check.monthlySavings} />
      {/if}
    </h2>
    <p class="mt-1 flex items-center gap-2 text-sm {style.ink}">
      <Icon name={style.icon} size={16} />{verdict.label}{#if check.verdict === 'short'}: short by <Money
          value={check.gap}
        /> a month{/if}.
    </p>
    <p class="muted mt-1 text-xs">
      Savings are the average of income minus spending over the last {plan.monthlySavings.months}
      {plan.monthlySavings.months === 1 ? 'month' : 'months'}, money lent left out.
    </p>
  </section>

  {#if check.tradeOffs.length}
    <Card title="Ways to fit the plan" subtitle="Each change alone closes the gap; lowest priority first" class="mt-4">
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each check.tradeOffs as tradeOff (tradeOff.goalId)}
          <li class="flex flex-wrap items-center justify-between gap-2 py-2">
            <span class="font-medium">{tradeOff.name}</span>
            <span class="muted">
              {#if tradeOff.moveTargetDateTo}Move from {formatMonth(tradeOff.targetDate)} to {formatMonth(
                  tradeOff.moveTargetDateTo,
                )}{:else}Moving the date alone is not enough{/if}
              ·
              {#if tradeOff.lowerTargetBy !== null}Lower the target by <Money
                  value={tradeOff.lowerTargetBy}
                />{:else}Lowering the target alone is not enough{/if}
            </span>
          </li>
        {/each}
      </ul>
    </Card>
  {/if}

  {#if plan.issues.length}
    <Card title="Rows to fix in the Goal Plan tab" class="mt-4">
      <ul class="space-y-1 text-sm">
        {#each plan.issues as issue (issue.row)}
          <li class="flex items-start gap-2 {STATUS.warning.ink}">
            <Icon name={STATUS.warning.icon} size={14} /> Row {issue.row}: {issue.message}
          </li>
        {/each}
      </ul>
    </Card>
  {/if}

  <div class="mt-4 grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
    <div>
      {#if groups.active.length}
        <ul class="grid gap-4 sm:grid-cols-2">
          {#each groups.active as goal, index (goal.id)}
            <GoalCard
              {goal}
              {busy}
              onup={index > 0 ? () => move(index, -1) : undefined}
              ondown={index < groups.active.length - 1 ? () => move(index, 1) : undefined}
              onedit={() => openEdit(goal)}
              ondelete={() => (deleting = goal)}
            />
          {/each}
        </ul>
      {:else}
        <Card><EmptyState title="No active goals" detail="Add a goal to see what it needs each month." /></Card>
      {/if}

      {#each [{ label: 'Paused', goals: groups.paused }, { label: 'Done', goals: groups.done }] as section (section.label)}
        {#if section.goals.length}
          <details class="glass mt-4 p-5">
            <summary class="cursor-pointer text-sm font-semibold">{section.label} ({section.goals.length})</summary>
            <ul class="mt-4 grid gap-4 sm:grid-cols-2">
              {#each section.goals as goal (goal.id)}
                <GoalCard {goal} {busy} onedit={() => openEdit(goal)} ondelete={() => (deleting = goal)} />
              {/each}
            </ul>
          </details>
        {/if}
      {/each}
    </div>

    <div class="space-y-4">
      <PoolCard pool={plan.pool} asOf={plan.asOf} />
      <Stat label="Left after goals" value={plan.unallocated} hint="Pool money no active goal needs now" />
      <Stat
        label="Monthly needs"
        value={plan.needs.perMonth}
        hint={plan.needs.fromTotalExpense
          ? 'Total spending: tag fixed and essential categories on the Budget page for needs only'
          : `Fixed and essential spending, ${plan.needs.months}-month average`}
      />
    </div>
  </div>
{/if}

<GoalFormModal
  open={creating || editing !== null}
  {editing}
  bind:form
  {busy}
  emergencyMonths={plan.settings.emergencyMonths}
  onclose={() => ((creating = false), (editing = null))}
  onsubmit={submit}
/>

<Modal open={deleting !== null} title="Delete goal" onclose={() => (deleting = null)}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? Its row in the Goal Plan tab is cleared. To keep it, mark it done or
      pause it instead.
    </p>
  {/if}
  {#snippet footer()}
    {#if deleting && deleting.status !== 'done'}
      {@const goal = deleting}
      <button
        type="button"
        class="btn-ghost"
        disabled={busy}
        onclick={() => setStatus(goal, 'done', `${goal.name} marked done.`)}>Mark done</button
      >
    {/if}
    <button type="button" class="btn-ghost" onclick={() => (deleting = null)}>Cancel</button>
    <button
      type="button"
      class="btn-danger"
      disabled={busy}
      onclick={() => {
        const goal = deleting;
        if (goal)
          void run(() => api.deleteGoal(goal.id, goal.etag), `${goal.name} deleted.`, 'Could not delete the goal');
      }}>{busy ? 'Deleting…' : 'Delete'}</button
    >
  {/snippet}
</Modal>
