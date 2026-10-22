<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import {
    track,
    Button,
    Card,
    Deferred,
    EmptyState,
    Icon,
    Modal,
    SkeletonCard,
    SkeletonHeadline,
    SkeletonStat,
    action,
  } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import type { GoalInput, GoalPlan, GoalProgress } from '$lib/api/types';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
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

  // The header renders at once; the plan streams in. New goal waits for it,
  // since a plan that is not set up yet offers Set up instead.
  let { data } = $props();
  const plan = track(() => data.plan);

  let creating = $state(false);
  let editing = $state<GoalProgress | null>(null);
  let deleting = $state<GoalProgress | null>(null);
  let form = $state(emptyGoalForm());
  // The goal whose arrow was pressed, so only that arrow spins.
  let moving = $state<{ id: string; direction: 'up' | 'down' } | null>(null);

  // A finished change closes the dialogs and reloads the plan; the shown plan
  // stays (with a Refreshing badge) until the new one arrives.
  async function done(message: string) {
    toasts.success(message);
    creating = false;
    editing = null;
    deleting = null;
    await invalidateAll();
  }

  const setup = action(
    async () => {
      await api.setupGoalPlan();
      await done('Goal plan created.');
    },
    { errorContext: 'Could not set up the goal plan' },
  );
  const create = action(
    async (input: GoalInput) => {
      await api.createGoal(input);
      await done(`${input.name} added.`);
    },
    { errorContext: 'Could not add the goal' },
  );
  const update = action(
    async (goal: GoalProgress, input: GoalInput, message: string) => {
      await api.updateGoal(goal.id, goal.etag, input);
      await done(message);
    },
    { errorContext: 'Could not save the goal' },
  );
  const remove = action(
    async (goal: GoalProgress) => {
      await api.deleteGoal(goal.id, goal.etag);
      await done(`${goal.name} deleted.`);
    },
    { errorContext: 'Could not delete the goal' },
  );
  const reorder = action(
    async (ids: string[]) => {
      try {
        await api.reorderGoals(ids);
        await done('Priorities updated.');
      } finally {
        moving = null;
      }
    },
    { errorContext: 'Could not reorder the goals' },
  );
  // One change at a time: each writes the Goal Plan tab with etags.
  const busy = $derived(setup.busy || create.busy || update.busy || remove.busy || reorder.busy);

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
    if (current) void update.run(current, input, `${input.name} saved.`);
    else void create.run(input);
  }

  // Reorders active goals; paused and done goals keep their places after them.
  function move(value: GoalPlan, index: number, direction: 'up' | 'down') {
    const groups = groupGoals(value);
    const active = moveId(
      groups.active.map((goal) => goal.id),
      index,
      direction === 'up' ? -1 : 1,
    );
    const ids = [...active, ...groups.paused.map((goal) => goal.id), ...groups.done.map((goal) => goal.id)];
    moving = { id: groups.active[index].id, direction };
    void reorder.run(ids);
  }

  function markDone(goal: GoalProgress) {
    const { input } = goalInputFrom({ ...goalFormFor(goal), status: 'done' }, goal.emergencyFund);
    if (input) void update.run(goal, input, `${goal.name} marked done.`);
  }
</script>

<PageHeader title="Goals" subtitle="Goals share one savings pool, filled in priority order">
  {#snippet actions()}
    {#if !plan.value?.setupNeeded}
      <Button variant="primary" icon="plus" disabled={!plan.value} onclick={openCreate}>New goal</Button>
    {/if}
  {/snippet}
</PageHeader>

<Deferred data={data.plan} label="the goal plan">
  {#snippet children(value)}
    {@const groups = groupGoals(value)}
    {@const check = value.check}
    {@const verdict = verdictStatus(check.verdict)}
    <div>
      {#if value.setupNeeded}
        <Card>
          <EmptyState
            title="Set up your goal plan"
            detail="This adds a Goal Plan tab to your sheet with an emergency fund first{value.legacyGoals
              ? ` and the ${value.legacyGoals} goals from your Goals tab`
              : ''}. Your existing tabs are not changed."
          />
          <p class="text-center">
            <Button variant="primary" busy={setup.busy} busyText="Setting up…" onclick={() => void setup.run()}
              >Set up your goal plan</Button
            >
          </p>
        </Card>
        <div class="mt-4 max-w-xl"><PoolCard pool={value.pool} asOf={value.asOf} /></div>
      {:else}
        <!-- Headline (ROADMAP P2): am I on track? -->
        <PageHeadline question="Am I on track?" tone={verdict.tone} id="goals-verdict">
          {#snippet answer()}
            <span>
              {#if check.verdict === 'no_deadlines'}
                You save <Money value={check.monthlySavings} /> a month
              {:else}
                Your goals need <Money value={check.requiredPerMonth} /> a month, you save
                <Money value={check.monthlySavings} />
              {/if}
            </span>
          {/snippet}
          <p class="font-medium {STATUS[verdict.tone].ink}">
            {verdict.label}{#if check.verdict === 'short'}: short by <Money value={check.gap} /> a month{/if}.
          </p>
          <p class="muted mt-1 text-xs">
            Savings are the average of income minus spending over the last {value.monthlySavings.months}
            {value.monthlySavings.months === 1 ? 'month' : 'months'}, money lent left out.
          </p>
        </PageHeadline>

        {#if check.tradeOffs.length}
          <Card
            title="Ways to fit the plan"
            subtitle="Each change alone closes the gap; lowest priority first"
            class="mt-4"
          >
            <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5">
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

        {#if value.issues.length}
          <Card title="Rows to fix in the Goal Plan tab" class="mt-4">
            <ul class="space-y-1 text-sm">
              {#each value.issues as issue (issue.row)}
                <li class="flex items-start gap-2 {STATUS.warning.ink}">
                  <Icon name={STATUS.warning.icon} size={14} /> Row {issue.row}: <MaskedText text={issue.message} />
                </li>
              {/each}
            </ul>
          </Card>
        {/if}

        <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
          <div>
            {#if groups.active.length}
              <ul class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {#each groups.active as goal, index (goal.id)}
                  <GoalCard
                    {goal}
                    {busy}
                    moving={moving?.id === goal.id ? moving.direction : null}
                    onup={index > 0 ? () => move(value, index, 'up') : undefined}
                    ondown={index < groups.active.length - 1 ? () => move(value, index, 'down') : undefined}
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
                <details class="panel mt-4 p-5">
                  <summary
                    class="cursor-pointer text-sm font-semibold transition-colors hover:text-accent-600 dark:hover:text-accent-300"
                    >{section.label} ({section.goals.length})</summary
                  >
                  <ul class="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
                    {#each section.goals as goal (goal.id)}
                      <GoalCard {goal} {busy} onedit={() => openEdit(goal)} ondelete={() => (deleting = goal)} />
                    {/each}
                  </ul>
                </details>
              {/if}
            {/each}
          </div>

          <div class="space-y-4">
            <PoolCard pool={value.pool} asOf={value.asOf} />
            <Stat label="Left after goals" value={value.unallocated} hint="Pool money no active goal needs now" />
            <Stat
              label="Monthly needs"
              value={value.needs.perMonth}
              hint={value.needs.fromTotalExpense
                ? 'Total spending: tag fixed and essential categories on the Budget page for needs only'
                : `Fixed and essential spending, ${value.needs.months}-month average`}
            />
          </div>
        </div>
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonHeadline question="Am I on track?" label="the goal plan" />
      <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <SkeletonCard lines={4} label="goals" />
          <SkeletonCard lines={4} label="goals" />
        </div>
        <div class="space-y-4">
          <SkeletonCard title="Savings pool" lines={4} />
          <SkeletonStat label="left after goals" />
        </div>
      </div>
    </div>
  {/snippet}
</Deferred>

<GoalFormModal
  open={creating || editing !== null}
  {editing}
  bind:form
  busy={create.busy || update.busy}
  emergencyMonths={plan.value?.settings.emergencyMonths ?? 0}
  onclose={() => ((creating = false), (editing = null))}
  onsubmit={submit}
/>

<Modal open={deleting !== null} title="Delete goal" onclose={() => (deleting = null)} {busy}>
  {#if deleting}
    <p class="text-sm">
      Delete <strong>{deleting.name}</strong>? Its row in the Goal Plan tab is cleared. To keep it, mark it done or
      pause it instead.
    </p>
  {/if}
  {#snippet footer()}
    {#if deleting && deleting.status !== 'done'}
      {@const goal = deleting}
      <Button busy={update.busy} busyText="Saving…" disabled={busy && !update.busy} onclick={() => markDone(goal)}
        >Mark done</Button
      >
    {/if}
    <Button disabled={busy} onclick={() => (deleting = null)}>Cancel</Button>
    <Button
      variant="danger"
      busy={remove.busy}
      busyText="Deleting…"
      disabled={busy && !remove.busy}
      onclick={() => {
        if (deleting) void remove.run(deleting);
      }}>Delete</Button
    >
  {/snippet}
</Modal>
