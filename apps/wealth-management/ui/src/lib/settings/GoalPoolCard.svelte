<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Preferences } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { STATUS } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';
  import { GOAL_POOL_FIELDS, goalPoolChanges, goalPoolError, type GoalPoolDraft } from './goalPool';

  // Goal pool settings (WM-SET-8) live in the Preferences tab of the sheet,
  // not in the engine settings store, so they save on their own.
  let prefs = $state<Preferences | null>(null);
  let loadError = $state<string | null>(null);
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
      loadError = error instanceof Error ? error.message : String(error);
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

<Card title="Goal pool" subtitle="Saved in the Preferences tab of your sheet; used by the goal plan">
  {#if loadError}
    <p class="text-sm {STATUS.critical.ink}">Preferences unavailable: {loadError}</p>
    <button type="button" class="btn-ghost mt-3" onclick={load}>Try again</button>
  {:else if !prefs}
    <p class="muted text-sm">Loading preferences…</p>
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
    <form class="-my-3 divide-y divide-slate-900/5 dark:divide-white/5" onsubmit={save}>
      {#each GOAL_POOL_FIELDS as field (field.key)}
        <div class="grid gap-x-6 gap-y-1.5 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center">
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
              bind:value={draft[field.key]}
            />
            <span class="muted text-xs">months (default {field.fallback})</span>
          </div>
        </div>
      {/each}
      <div class="flex items-center justify-end gap-3 py-3">
        {#if problem}<span class="text-xs {STATUS.warning.ink}">{problem}</span>{/if}
        <button type="submit" class="btn-primary" disabled={saving || !!problem || !changes.length}
          >{saving ? 'Saving...' : 'Save'}</button
        >
      </div>
    </form>
  {/if}
</Card>
