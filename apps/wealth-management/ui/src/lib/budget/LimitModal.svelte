<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { BudgetLine } from '$lib/api/types';
  import { Button, Modal } from '@libs/ui-kit';
  import { formatMonth } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';

  // Limit editing (WM-BUD-6). The page seeds scope and value when it opens the editor.
  interface Props {
    line: BudgetLine | null;
    month: string;
    scope: 'month' | 'year';
    value: number | null;
    onclose: () => void;
  }

  let { line, month, scope = $bindable(), value = $bindable(), onclose }: Props = $props();
  let saving = $state(false);

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!line || value === null || value < 0) return;
    saving = true;
    try {
      await api.updateBudgetLimit({
        year: Number(month.slice(0, 4)),
        category: line.category,
        // An empty month writes the yearly limit column.
        month: scope === 'month' ? month : undefined,
        limit: value,
      });
      toasts.push('success', `Limit for ${line.category} updated.`);
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not update the limit');
    } finally {
      saving = false;
    }
  }
</script>

<Modal open={line !== null} title="Limit for {line?.category ?? ''}" {onclose} busy={saving}>
  <form id="limit-form" class="space-y-4" onsubmit={save}>
    <fieldset class="space-y-2 text-sm">
      <legend class="label">Apply to</legend>
      <label class="flex items-center gap-2"
        ><input
          type="radio"
          bind:group={scope}
          value="month"
          class="accent-accent-600"
          onchange={() => (value = line?.limit ?? 0)}
        />
        Monthly limit for {formatMonth(month)}</label
      >
      <label class="flex items-center gap-2"
        ><input
          type="radio"
          bind:group={scope}
          value="year"
          class="accent-accent-600"
          onchange={() => (value = line?.yearlyLimit ?? 0)}
        />
        Yearly limit for {month.slice(0, 4)}</label
      >
    </fieldset>
    <div>
      <label class="label" for="limit-value">Limit (VND)</label>
      <input id="limit-value" type="number" min="0" step="1000" class="field" bind:value required data-autofocus />
    </div>
  </form>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="limit-form" variant="primary" busy={saving} busyText="Saving…">Save limit</Button>
  {/snippet}
</Modal>
