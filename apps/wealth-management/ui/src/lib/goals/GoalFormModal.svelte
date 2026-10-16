<script lang="ts">
  import type { SubmitHandler } from '$lib/accounts/handlers';
  import type { GoalProgress } from '$lib/api/types';
  import Modal from '$lib/components/Modal.svelte';
  import { looksLikeEmergencyFund, type GoalForm } from './goals';

  interface Props {
    open: boolean;
    /** The goal being edited, or null when creating. */
    editing: GoalProgress | null;
    form: GoalForm;
    busy: boolean;
    emergencyMonths: number;
    onclose: () => void;
    onsubmit: SubmitHandler;
  }

  let { open, editing, form = $bindable(), busy, emergencyMonths, onclose, onsubmit }: Props = $props();

  const emergency = $derived(editing?.emergencyFund || looksLikeEmergencyFund(form.name, form.note));
</script>

<Modal {open} title={editing ? `Edit ${editing.name}` : 'New goal'} {onclose}>
  <form id="goal-form" class="grid gap-4 sm:grid-cols-2" {onsubmit}>
    <div class="sm:col-span-2">
      <label class="label" for="goal-name">Name *</label>
      <input id="goal-name" class="field" bind:value={form.name} required />
    </div>
    <div>
      <label class="label" for="goal-target">Target (VND){emergency ? '' : ' *'}</label>
      <input
        id="goal-target"
        type="number"
        min="1"
        step="1"
        class="field"
        bind:value={form.target}
        aria-describedby="goal-target-hint"
      />
      <p id="goal-target-hint" class="muted mt-1 text-xs">
        {#if emergency}
          Leave blank to use {emergencyMonths} months of your needs.
        {:else}
          The amount the goal needs in total.
        {/if}
      </p>
    </div>
    <div>
      <label class="label" for="goal-date">Target date</label>
      <input id="goal-date" type="month" class="field" bind:value={form.targetDate} aria-describedby="goal-date-hint" />
      <p id="goal-date-hint" class="muted mt-1 text-xs">Blank: no deadline, the pace only.</p>
    </div>
    <div>
      <label class="label" for="goal-priority">Priority</label>
      <input
        id="goal-priority"
        type="number"
        min="1"
        step="1"
        class="field"
        bind:value={form.priority}
        aria-describedby="goal-priority-hint"
      />
      <p id="goal-priority-hint" class="muted mt-1 text-xs">1 is filled first. Blank puts a new goal last.</p>
    </div>
    <div>
      <label class="label" for="goal-starting">Starting amount (VND)</label>
      <input
        id="goal-starting"
        type="number"
        min="0"
        step="1"
        class="field"
        bind:value={form.startingAmount}
        aria-describedby="goal-starting-hint"
      />
      <p id="goal-starting-hint" class="muted mt-1 text-xs">Money already set aside outside your accounts.</p>
    </div>
    <div>
      <label class="label" for="goal-status">Status</label>
      <select id="goal-status" class="field" bind:value={form.status}>
        <option value="active">Active</option>
        <option value="paused">Paused</option>
        <option value="done">Done</option>
      </select>
    </div>
    <div class="sm:col-span-2">
      <label class="label" for="goal-note">Note</label>
      <input id="goal-note" class="field" bind:value={form.note} />
    </div>
  </form>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={onclose}>Cancel</button>
    <button type="submit" form="goal-form" class="btn-primary" disabled={busy}
      >{busy ? 'Saving…' : editing ? 'Save' : 'Create'}</button
    >
  {/snippet}
</Modal>
