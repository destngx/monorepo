<script lang="ts">
  import type { GoalProgress } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatMonth, formatPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { finishLabel, formatYears, paceStatus } from './goals';

  interface Props {
    goal: GoalProgress;
    busy: boolean;
    /** Reorder handlers; absent hides the arrows (paused and done goals). */
    onup?: () => void;
    ondown?: () => void;
    onedit: () => void;
    ondelete: () => void;
  }

  let { goal, busy, onup, ondown, onedit, ondelete }: Props = $props();

  const status = $derived(paceStatus(goal.pace));
</script>

<li class="glass p-4">
  <div class="flex items-start justify-between gap-3">
    <div class="min-w-0">
      <p class="truncate font-semibold">
        <span class="muted mr-1 text-xs font-medium">#{goal.priority}</span>{goal.name}
      </p>
      <div class="mt-1 flex flex-wrap items-center gap-1.5">
        <Badge status={status.tone}>{status.label}</Badge>
        {#if goal.emergencyFund}<Badge tone="violet">Emergency fund</Badge>{/if}
      </div>
    </div>
    <div class="flex shrink-0">
      {#if onup}
        <button type="button" class="btn-ghost px-2" aria-label="Move {goal.name} up" disabled={busy} onclick={onup}
          ><span class="inline-block rotate-180"><Icon name="chevronDown" size={15} /></span></button
        >
      {/if}
      {#if ondown}
        <button type="button" class="btn-ghost px-2" aria-label="Move {goal.name} down" disabled={busy} onclick={ondown}
          ><Icon name="chevronDown" size={15} /></button
        >
      {/if}
      <button type="button" class="btn-ghost px-2" aria-label="Edit {goal.name}" onclick={onedit}
        ><Icon name="pencil" size={15} /></button
      >
      <button type="button" class="btn-ghost px-2" aria-label="Delete {goal.name}" onclick={ondelete}
        ><Icon name="trash" size={15} /></button
      >
    </div>
  </div>

  <p class="mt-3 text-sm">
    <Money value={goal.funded} class="font-semibold" /> <span class="muted">of</span>
    <Money value={goal.targetAmount} />
    {#if goal.targetComputed}<span class="muted text-xs">(from your monthly needs)</span>{/if}
  </p>
  {#if goal.longHorizon && goal.pace !== 'done' && goal.pace !== 'paused'}
    <p class="mt-2 text-lg font-semibold">
      {goal.yearsToReach === null ? 'Not reachable at this pace' : `${formatYears(goal.yearsToReach)} to reach`}
    </p>
    <p class="muted text-xs">{formatPercent(goal.percent, 0)} funded so far</p>
  {:else}
    <div class="mt-2 flex items-center gap-2">
      <ProgressBar percent={goal.percent} color={STATUS[status.tone].fill} label="{goal.name} funded" />
      <span class="muted text-xs">{formatPercent(goal.percent, 0)}</span>
    </div>
  {/if}

  <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
    <dt class="muted">Target date</dt>
    <dd class="text-right">{goal.targetDate ? formatMonth(goal.targetDate) : 'None'}</dd>
    {#if goal.requiredPerMonth !== null}
      <dt class="muted">Needs per month</dt>
      <dd class="text-right"><Money value={goal.requiredPerMonth} /></dd>
    {/if}
    <dt class="muted">Projected finish</dt>
    <dd class="text-right">{finishLabel(goal)}</dd>
    {#if goal.startingAmount > 0}
      <dt class="muted">Set aside outside the pool</dt>
      <dd class="text-right"><Money value={goal.startingAmount} /></dd>
    {/if}
  </dl>
  {#if goal.note}<p class="muted mt-2 truncate text-xs">{goal.note}</p>{/if}
</li>
