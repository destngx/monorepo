<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Allocation, AllocationTargets } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { targetAllocation } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import {
    ALLOCATION_COLORS,
    allocationStatus,
    breachSummary,
    lineStatus,
    parseBrowserTargets,
    TARGET_CLASSES,
    targetsError,
    targetValues,
  } from './allocation';

  // The full allocation (ROADMAP C4, WM-INV-20): engine figures, targets
  // from the Preferences tab (C6) and drift. Accounts and Health show a
  // compact view of the same answer.
  let { allocation, onchange }: { allocation: Allocation; onchange: (next: Allocation) => void } = $props();

  const status = $derived(allocationStatus(allocation));
  const breaches = $derived(breachSummary(allocation));

  // Targets an older version kept in this browser: offered once for the sheet.
  const browserTargets = $derived(parseBrowserTargets(targetAllocation.value));
  const offerMove = $derived(browserTargets !== null && allocation.targets === null);
  $effect(() => {
    // The sheet is the store once it has targets; drop the browser copy.
    if (allocation.targets && targetAllocation.value) targetAllocation.value = '';
  });

  let editing = $state(false);
  let saving = $state(false);
  let draft = $state<AllocationTargets>({ cash: 0, funds: 0, crypto: 0 });

  function edit() {
    draft = allocation.targets ? { ...allocation.targets } : { cash: 0, funds: 0, crypto: 0 };
    editing = true;
  }

  async function save(targets: AllocationTargets | null, success: string) {
    saving = true;
    try {
      await api.savePreferences(targetValues(targets));
      targetAllocation.value = '';
      onchange(await api.allocation());
      editing = false;
      toasts.push('success', success);
    } catch (error) {
      toasts.error(error, 'Could not save the targets');
    } finally {
      saving = false;
    }
  }

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const next = { cash: Number(draft.cash), funds: Number(draft.funds), crypto: Number(draft.crypto) };
    const problem = targetsError(next);
    if (problem) {
      toasts.push('warning', problem);
      return;
    }
    void save(next, 'Targets saved to your sheet.');
  }

  const describe = (targets: AllocationTargets) =>
    TARGET_CLASSES.map(({ key, label }) => `${label} ${formatPercent(targets[key], 0)}`).join(', ');
</script>

<Card
  title="Allocation"
  subtitle="Rebalance when a class drifts more than {allocation.driftThreshold} points from its target; money lent has no target"
  class="mt-4"
>
  {#snippet actions()}
    <Badge status={status.tone}>{status.label}</Badge>
    {#if !editing}
      <button type="button" class="btn-ghost text-xs" onclick={edit}
        ><Icon name="pencil" size={14} /> {allocation.targets ? 'Edit targets' : 'Set targets'}</button
      >
    {/if}
  {/snippet}

  {#if offerMove && browserTargets}
    <div class="mb-3 flex flex-wrap items-center gap-3 rounded-xl border p-3 text-sm {STATUS.info.box}" role="status">
      <Icon name={STATUS.info.icon} size={16} />
      <span class="min-w-0 flex-1"
        >Targets saved in this browser only: {describe(browserTargets)}. Save them to your sheet so every device uses
        them.</span
      >
      <button
        type="button"
        class="btn-primary text-xs"
        disabled={saving}
        onclick={() => save(browserTargets, 'Targets moved to your sheet.')}>Save to your sheet</button
      >
      <button type="button" class="btn-ghost text-xs" disabled={saving} onclick={() => (targetAllocation.value = '')}
        >Discard</button
      >
    </div>
  {/if}

  {#if editing}
    <form class="mb-4 grid gap-3 sm:grid-cols-4 sm:items-end" onsubmit={submit}>
      {#each TARGET_CLASSES as targetClass (targetClass.key)}
        <div>
          <label class="label" for="target-{targetClass.key}">{targetClass.label} (%)</label>
          <input
            id="target-{targetClass.key}"
            type="number"
            min="0"
            max="100"
            step="0.5"
            class="field"
            bind:value={draft[targetClass.key]}
          />
        </div>
      {/each}
      <div class="flex gap-2">
        <button type="submit" class="btn-primary" disabled={saving}>Save</button>
        <button type="button" class="btn-ghost" onclick={() => (editing = false)}>Cancel</button>
        {#if allocation.targets}<button
            type="button"
            class="btn-ghost"
            disabled={saving}
            onclick={() => save(null, 'Targets cleared.')}>Clear</button
          >{/if}
      </div>
    </form>
  {/if}

  {#if breaches.length}
    <div class="mb-3 flex items-start gap-2 rounded-xl border p-3 text-sm {STATUS.warning.box}" role="alert">
      <Icon name={STATUS.warning.icon} size={16} class="mt-0.5" />
      <span>Rebalance suggested: {breaches.join('; ')}.</span>
    </div>
  {/if}

  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[40rem] text-sm">
      <thead>
        <tr class="muted text-left text-xs uppercase">
          <th class="px-5 py-2">Class</th>
          <th class="px-5 py-2 text-right">Value</th>
          <th class="px-5 py-2 text-right" title="Share of everything allocated, money lent included">Share</th>
          <th class="px-5 py-2 text-right" title="Share of cash, funds and crypto, compared with the target">Current</th
          >
          <th class="px-5 py-2 text-right">Target</th>
          <th class="px-5 py-2 text-right">Drift</th>
          <th class="px-5 py-2">Status</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each allocation.lines as line (line.class)}
          {@const lineState = lineStatus(line.status)}
          <tr>
            <td class="px-5 py-2">
              <span class="flex items-center gap-2">
                <span class="size-2.5 shrink-0 rounded-full" style:background-color={ALLOCATION_COLORS[line.class]}
                ></span>{line.label}
              </span>
            </td>
            <td class="px-5 py-2 text-right"><Money value={line.value} /></td>
            <td class="tabular px-5 py-2 text-right">{formatPercent(line.share)}</td>
            <td class="tabular px-5 py-2 text-right">{formatPercent(line.current)}</td>
            <td class="tabular px-5 py-2 text-right">{formatPercent(line.target)}</td>
            <td class="tabular px-5 py-2 text-right"
              >{line.drift === null ? '-' : `${line.drift > 0 ? '+' : ''}${line.drift.toFixed(1)} pts`}</td
            >
            <td class="px-5 py-2"
              ><Badge status={lineState.tone}>{line.targeted ? lineState.label : 'Not targeted'}</Badge></td
            >
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
  {#if allocation.unvalued}
    <p class="muted mt-2 text-xs">{allocation.unvalued} holdings are not valued and are not counted.</p>
  {/if}
  {#if !allocation.targets && !offerMove}
    <p class="muted mt-2 text-xs">
      Set a target share for cash, fund certificates and crypto to track drift. Targets are saved in the Preferences tab
      of your sheet.
    </p>
  {/if}
</Card>
