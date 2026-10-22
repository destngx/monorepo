<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import { classifyFailure, type LoadFailure } from '$lib/api/load';
  import type { Preferences } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import DayToDayFloatField from '$lib/bills/DayToDayFloatField.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import LoadError from '$lib/components/LoadError.svelte';
  import SkeletonText from '$lib/components/skeleton/SkeletonText.svelte';
  import { STATUS } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';
  import { GOAL_POOL_FIELDS, goalPoolChanges, goalPoolError, type GoalPoolDraft } from './goalPool';

  // Goal pool settings (WM-SET-8) and the idle cash day-to-day float
  // (WM-SET-11) live in the Preferences tab of the sheet, not in the engine
  // settings store, so they save on their own.
  let prefs = $state<Preferences | null>(null);
  let loadError = $state<LoadFailure | null>(null);
  let draft = $state<GoalPoolDraft>({ emergencyMonths: '', spendingBufferMonths: '' });
  let saving = $state(false);

  async function load() {
    loadError = null;
    try {
      prefs = await api.preferences();
      draft = {
        emergencyMonths: String(prefs.goalPool.emergencyMonths),
        spendingBufferMonths: String(prefs.goalPool.spendingBufferMonths),
      };
    } catch (error) {
      loadError = classifyFailure(error);
    }
  }

  $effect(() => {
    void load();
  });

  const problem = $derived(goalPoolError(draft));
  const changes = $derived(prefs ? goalPoolChanges(prefs.goalPool, draft) : []);

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (problem || !changes.length) return;
    saving = true;
    try {
      prefs = await api.savePreferences(changes);
      toasts.push('success', 'Goal settings saved to your sheet.');
    } catch (error) {
      toasts.error(error, 'Could not save the goal settings');
    } finally {
      saving = false;
    }
  }
</script>

<Card title="Goal pool" subtitle="Saved in the Preferences tab of your sheet; used by the goal plan and idle cash">
  {#if loadError && !prefs}
    <LoadError failure={loadError} label="the goal settings" retry={load} />
  {:else if !prefs}
    <SkeletonText lines={4} label="the goal settings" />
  {:else}
    {#if prefs.issues.length}
      <div class="mb-3 rounded-xl border p-3 text-xs {STATUS.warning.box}" role="status">
        <p class="flex items-center gap-2 font-medium">
          <Icon name={STATUS.warning.icon} size={14} />Some rows of the Preferences tab are ignored:
        </p>
        <ul class="mt-1 list-disc pl-5">
          {#each prefs.issues as issue, i (i)}<li>{issue}</li>{/each}
        </ul>
      </div>
    {/if}
    <form class="-my-3 divide-y divide-stone-900/5 dark:divide-white/5" onsubmit={save}>
      {#each GOAL_POOL_FIELDS as field (field.key)}
        <div class="grid grid-cols-1 gap-x-6 gap-y-1.5 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center">
          <div>
            <label class="text-sm font-medium" for="goal-{field.key}">{field.label}</label>
            <p class="muted text-xs">{field.hint}</p>
          </div>
          <div class="flex items-center gap-2">
            <input
              id="goal-{field.key}"
              class="field w-32"
              type="number"
              min={field.min}
              max={field.max}
              step="0.5"
              aria-invalid={problem ? true : undefined}
              aria-describedby={problem ? 'goal-pool-problem' : undefined}
              bind:value={draft[field.key]}
            />
            <span class="muted text-xs">months (default {field.fallback})</span>
          </div>
        </div>
      {/each}
      <div class="flex items-center justify-end gap-3 py-3">
        {#if problem}<span id="goal-pool-problem" class="text-xs {STATUS.warning.ink}" role="alert">{problem}</span
          >{/if}
        <Button type="submit" variant="primary" busy={saving} busyText="Saving…" disabled={!!problem || !changes.length}
          >Save</Button
        >
      </div>
    </form>
    <div class="border-t border-stone-900/5 pt-3 dark:border-white/5">
      <p class="text-sm font-medium">Idle cash</p>
      <p class="muted mb-1 text-xs">
        Money kept ready in everyday accounts before any cash counts as idle. Emergency accounts are marked on Bills.
      </p>
      <DayToDayFloatField
        saved={prefs.idleCash.dayToDayFloat}
        defaultHint="the spending buffer, spending buffer months x average monthly expense"
        onsaved={(next) => (prefs = next)}
      />
    </div>
  {/if}
</Card>
