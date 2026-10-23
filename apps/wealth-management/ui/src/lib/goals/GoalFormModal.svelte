<script lang="ts">
  import type { SubmitHandler } from '$lib/accounts/handlers';
  import type { GoalProgress } from '$lib/api/types';
  import { Button, Modal } from '@libs/ui-kit';
  import MonthPicker from '$lib/components/MonthPicker.svelte';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { goalDateHint, looksLikeEmergencyFund, type GoalForm } from './goals';

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

  // Fields show as invalid once a submit was tried (or the name was left blank).
  let tried = $state(false);
  let nameTouched = $state(false);
  const nameInvalid = $derived((tried || nameTouched) && !form.name.trim());
  const targetInvalid = $derived(tried && (form.target === null ? !emergency : form.target <= 0));
  // Fields AI filled from the Describe it box (WM-FILL-6).
  const marks = new FillMarks();
  $effect(() => {
    if (!open) {
      tried = false;
      nameTouched = false;
      marks.clear();
    }
  });

  function submit(event: SubmitEvent) {
    tried = true;
    onsubmit(event);
  }
</script>

<Modal {open} title={editing ? `Edit ${editing.name}` : 'New goal'} {onclose} {busy}>
  <FillBox
    form="goal"
    class="mb-4"
    current={() => ({ ...form })}
    onfill={(result) => {
      form = applyFill(form, result.values);
      marks.take(result);
    }}
  />
  <form id="goal-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={submit}>
    <div class="sm:col-span-2">
      <label class="label" for="goal-name">Name *<FilledMark show={marks.has('name', form.name)} /></label>
      <input
        id="goal-name"
        class="field"
        bind:value={form.name}
        required
        aria-invalid={nameInvalid}
        aria-describedby={nameInvalid ? 'goal-name-error' : undefined}
        onblur={() => (nameTouched = true)}
        oninvalid={() => (tried = true)}
      />
      {#if nameInvalid}
        <p id="goal-name-error" class="mt-1 text-xs text-rose-600 dark:text-rose-400">Enter a name for the goal.</p>
      {/if}
    </div>
    <div>
      <label class="label" for="goal-target"
        >Target (VND){emergency ? '' : ' *'}<FilledMark show={marks.has('target', form.target)} /></label
      >
      <input
        id="goal-target"
        type="number"
        min="1"
        step="1"
        class="field"
        bind:value={form.target}
        aria-invalid={targetInvalid}
        aria-describedby="goal-target-hint"
        oninvalid={() => (tried = true)}
      />
      <p id="goal-target-hint" class="mt-1 text-xs {targetInvalid ? 'text-rose-600 dark:text-rose-400' : 'muted'}">
        {#if targetInvalid}
          {form.target === null ? 'Enter the amount the goal needs.' : 'The target must be more than 0.'}
        {:else if emergency}
          Leave blank to use {emergencyMonths} months of your needs.
        {:else}
          The amount the goal needs in total.
        {/if}
      </p>
    </div>
    <div>
      <label class="label" for="goal-date"
        >Target date<FilledMark show={marks.has('targetDate', form.targetDate)} /></label
      >
      <MonthPicker
        id="goal-date"
        bind:value={form.targetDate}
        placeholder="No deadline"
        clearLabel="No deadline"
        describedby="goal-date-hint"
      />
      <p id="goal-date-hint" class="muted mt-1 text-xs">{goalDateHint(form.targetDate)}</p>
    </div>
    <div>
      <label class="label" for="goal-priority">Priority<FilledMark show={marks.has('priority', form.priority)} /></label
      >
      <input
        id="goal-priority"
        type="number"
        min="1"
        step="1"
        class="field"
        bind:value={form.priority}
        aria-describedby="goal-priority-hint"
      />
      <p id="goal-priority-hint" class="muted mt-1 text-xs">
        1 is filled first; goals with the same priority share equally. Blank puts a new goal last.
      </p>
    </div>
    <div>
      <label class="label" for="goal-starting"
        >Starting amount (VND)<FilledMark show={marks.has('startingAmount', form.startingAmount)} /></label
      >
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
      <label class="label" for="goal-status">Status<FilledMark show={marks.has('status', form.status)} /></label>
      <select id="goal-status" class="field" bind:value={form.status}>
        <option value="active">Active</option>
        <option value="paused">Paused</option>
        <option value="done">Done</option>
      </select>
    </div>
    <div class="sm:col-span-2">
      <label class="label" for="goal-note">Note<FilledMark show={marks.has('note', form.note)} /></label>
      <input id="goal-note" class="field" bind:value={form.note} />
    </div>
    <div class="sm:col-span-2">
      <label class="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          class="size-4 accent-accent-600"
          bind:checked={form.useSlowMoney}
          aria-describedby="goal-slow-hint"
        />
        Can use slow money
      </label>
      <p id="goal-slow-hint" class="muted mt-1 text-xs">
        Investment and long holding accounts, and accounts marked not ready to use, take days to withdraw. Leave this
        off for money you may need fast, like the emergency fund.
      </p>
    </div>
  </form>
  {#snippet footer()}
    <Button disabled={busy} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="goal-form" variant="primary" {busy} busyText="Saving…"
      >{editing ? 'Save' : 'Create'}</Button
    >
  {/snippet}
</Modal>
