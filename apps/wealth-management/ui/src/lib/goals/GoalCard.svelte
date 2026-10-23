<script lang="ts">
  import type { GoalProgress } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button, Icon, Spinner } from '@libs/ui-kit';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatMonth, formatPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { finishLabel, formatYears, paceStatus } from './goals';

  interface Props {
    goal: GoalProgress;
    /** Another change to the plan is running: reorders wait for it. */
    busy: boolean;
    /** This goal is being moved: that arrow, or the drag handle, shows the spinner. */
    moving?: 'up' | 'down' | 'drag' | null;
    /** Reorder handlers; absent hides the arrows (paused and done goals). With either, the card shows a drag handle. */
    onup?: () => void;
    ondown?: () => void;
    onedit: () => void;
    ondelete: () => void;
  }

  let { goal, busy, moving = null, onup, ondown, onedit, ondelete }: Props = $props();

  const status = $derived(paceStatus(goal.pace));
</script>

<li
  class="panel p-4 data-dragging:relative data-dragging:z-20 data-dragging:cursor-grabbing data-dragging:shadow-xl data-drop-target:outline-2 data-drop-target:outline-offset-2 data-drop-target:outline-accent-500 data-drop-target:outline-dashed"
>
  <div class="flex items-start justify-between gap-3">
    {#if onup || ondown}
      <!-- Pointer drag only; the arrows are the keyboard way to reorder. -->
      <span
        data-drag-handle
        class="hit-area -ml-1 grid h-8 w-5 shrink-0 touch-none place-items-center text-ink-500 select-none {busy
          ? 'cursor-not-allowed opacity-50'
          : 'cursor-grab'}"
        title="Drag to reorder"
        aria-hidden="true"
      >
        {#if moving === 'drag'}<Spinner />{:else}<Icon name="grip" size={16} />{/if}
      </span>
    {/if}
    <div class="min-w-0 flex-1">
      <p class="truncate font-semibold">
        <span class="muted mr-1 text-xs font-medium">#{goal.priority}</span>{goal.name}
      </p>
      <div class="mt-1 flex flex-wrap items-center gap-1.5">
        <Badge status={status.tone}>{status.label}</Badge>
        {#if goal.emergencyFund}<Badge tone="violet">Emergency fund</Badge>{/if}
        {#if goal.useSlowMoney}<Badge>Uses slow money</Badge>{/if}
      </div>
    </div>
    <div class="flex shrink-0">
      {#if onup}
        <Button
          size="icon"
          icon="chevronDown"
          class="[&>svg]:rotate-180"
          label="Move {goal.name} up"
          busy={moving === 'up'}
          disabled={busy && moving !== 'up'}
          onclick={onup}
        />
      {/if}
      {#if ondown}
        <Button
          size="icon"
          icon="chevronDown"
          label="Move {goal.name} down"
          busy={moving === 'down'}
          disabled={busy && moving !== 'down'}
          onclick={ondown}
        />
      {/if}
      <Button size="icon" icon="pencil" label="Edit {goal.name}" onclick={onedit} />
      <Button size="icon" icon="trash" label="Delete {goal.name}" onclick={ondelete} />
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
    {#if goal.fromSlow > 0}
      <dt class="muted">From slow money</dt>
      <dd class="text-right"><Money value={goal.fromSlow} /></dd>
    {/if}
    {#if goal.startingAmount > 0}
      <dt class="muted">Set aside outside the pool</dt>
      <dd class="text-right"><Money value={goal.startingAmount} /></dd>
    {/if}
  </dl>
  {#if goal.note}<p class="muted mt-2 truncate text-xs"><MaskedText text={goal.note} /></p>{/if}
</li>
