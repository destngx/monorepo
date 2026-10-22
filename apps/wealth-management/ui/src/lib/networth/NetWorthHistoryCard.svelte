<script lang="ts">
  import { base } from '$app/paths';
  import type { NetWorthHistory } from '$lib/api/types';
  import { Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatMonth, formatSignedPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import {
    estimatedCount,
    milestonesInRange,
    milestoneText,
    NET_WORTH_RANGES,
    type NetWorthRange,
    pointsInRange,
    rangeChange,
    sourceLabel,
  } from './history';
  import NetWorthHistoryChart from './NetWorthHistoryChart.svelte';

  // ROADMAP P3: net worth over 1Y and all time, built from the monthly snapshots. Months without a
  // snapshot are estimated from transactions and drawn dashed with hollow markers.
  let { history }: { history: NetWorthHistory } = $props();

  let range = $state<NetWorthRange>('1Y');
  let showTable = $state(false);
  const points = $derived(pointsInRange(history.points, range));
  const change = $derived(rangeChange(points));
  const milestones = $derived(milestonesInRange(history.milestones, points));
  const estimated = $derived(estimatedCount(points));
  const rangeText = $derived(range === '1Y' ? 'over the last year' : `since ${formatMonth(points[0]?.month ?? '')}`);
</script>

<Card title="Net worth history" subtitle="Month-end net worth from your snapshots; other months are estimated">
  {#snippet actions()}
    <div class="flex gap-1" role="group" aria-label="Range">
      {#each NET_WORTH_RANGES as option (option)}
        <button
          type="button"
          class="btn-ghost rounded-md px-2 py-0.5 text-xs {range === option
            ? 'bg-accent-500/15 font-semibold text-accent-700 hover:bg-accent-500/20 dark:text-accent-300'
            : 'muted'}"
          title="Show {option === '1Y' ? 'the last year' : 'all time'}"
          aria-pressed={range === option}
          onclick={() => (range = option)}>{option}</button
        >
      {/each}
    </div>
  {/snippet}

  <div class="border-l-4 pl-3 {STATUS.info.edge}">
    <p class="text-lg" data-testid="networth-headline">
      Net worth <strong><Money value={history.current} /></strong>{#if change}; <Money value={change.change} signed />
        ({formatSignedPercent(change.percent)}) {rangeText}{#if change.estimated}, from an estimate{/if}{/if}.
    </p>
    {#if estimated > 0}
      <p class="muted mt-1 text-xs" data-testid="networth-estimated">
        {estimated} of {points.length} months are estimated from transactions (dashed line, hollow markers): they miss price
        moves. Write a snapshot at each
        <a class="link" href="{base}/close">monthly close</a> to measure them.
      </p>
    {/if}
  </div>

  <div class="mt-4">
    <NetWorthHistoryChart {points} {milestones} />
  </div>

  {#if milestones.length}
    <ul class="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-sm" aria-label="Milestones">
      {#each milestones as milestone (`${milestone.month}-${milestone.kind}`)}
        <li class="flex items-center gap-1.5">
          <span aria-hidden="true">◆</span>
          <span class="muted tabular">{formatMonth(milestone.month)}</span>
          <span
            >{milestoneText(milestone)}
            {#if milestone.kind === 'threshold'}<Money value={milestone.threshold} />{:else}<Money
                value={milestone.netWorth}
              />{/if}{#if milestone.estimated}<span class="muted"> (estimated)</span>{/if}</span
          >
        </li>
      {/each}
    </ul>
  {/if}

  <button
    type="button"
    class="btn-ghost mt-3 px-2 py-1 text-xs"
    aria-expanded={showTable}
    onclick={() => (showTable = !showTable)}>{showTable ? 'Hide the table' : 'Show as a table'}</button
  >
  {#if showTable}
    <div class="mt-2 overflow-x-auto">
      <table class="w-full text-sm" data-testid="networth-table">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="py-2">Month</th>
            <th class="py-2">Source</th>
            <th class="py-2 text-right">Net worth</th>
            <th class="py-2 text-right">Change</th>
            <th class="py-2 text-right">Savings</th>
            <th class="py-2 text-right">Market and other</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
          {#each [...points].reverse() as point (point.month)}
            <tr>
              <td class="py-2">{formatMonth(point.month)}</td>
              <td class="py-2 {point.estimated ? 'muted italic' : ''}">{sourceLabel(point)}</td>
              <td class="py-2 text-right"><Money value={point.netWorth} /></td>
              <td class="py-2 text-right">
                {#if point.change !== null}<Money value={point.change} signed />{:else}<span class="muted">-</span>{/if}
              </td>
              <td class="py-2 text-right">
                {#if point.savings !== null}<Money value={point.savings} signed />{:else}<span class="muted">-</span
                  >{/if}
              </td>
              <td class="py-2 text-right">
                {#if point.market !== null}<Money value={point.market} signed />{:else}<span class="muted">-</span>{/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    <p class="muted mt-2 text-xs">
      Savings is income minus expense between two measured months; market and other is the rest of the change: price
      moves, balance corrections and anything not recorded as income or expense. It is shown only when both months are
      measured.
    </p>
  {/if}
  {#if history.issues.length}
    <ul class="muted mt-2 list-disc pl-5 text-xs" aria-label="Snapshot rows ignored">
      {#each history.issues as issue (issue.row)}<li>Snapshots row {issue.row}: {issue.message}</li>{/each}
    </ul>
  {/if}
</Card>
