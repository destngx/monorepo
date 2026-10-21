<script lang="ts">
  import type { ObservationInput, Page } from '$lib/api/types';
  import { catalog } from '$lib/state/catalog.svelte';
  import { action } from '$lib/state/action.svelte';
  import Field from './Field.svelte';
  import Modal from './Modal.svelte';

  // Adds or edits one observation (HW-READ-4, HW-READ-7).
  interface Props {
    /** The values to edit; null closes the modal. */
    value: ObservationInput | null;
    title: string;
    pages: Page[];
    onsave: (input: ObservationInput) => Promise<unknown>;
    onclose: () => void;
  }

  let { value, title, pages, onsave, onclose }: Props = $props();
  let form = $state<ObservationInput>(blank());

  function blank(): ObservationInput {
    return { analyte: '', metricCode: '', value: '', unit: '', reference: '', flag: '', sampleDate: '', pageId: '' };
  }

  $effect(() => {
    if (value) form = { ...value };
  });

  const save = action(
    async () => {
      await onsave({ ...form });
      onclose();
    },
    { errorContext: 'Could not save', success: 'Saved' },
  );
</script>

<Modal open={value !== null} {title} {onclose}>
  <form
    id="obs-form"
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void save.run();
    }}
  >
    <Field id="obs-analyte" label="Test name, as printed">
      <input id="obs-analyte" class="field" bind:value={form.analyte} required maxlength="120" />
    </Field>
    <div class="grid grid-cols-2 gap-4">
      <Field id="obs-value" label="Result">
        <input id="obs-value" class="field" bind:value={form.value} required maxlength="60" />
      </Field>
      <Field id="obs-unit" label="Unit">
        <input id="obs-unit" class="field" bind:value={form.unit} maxlength="30" />
      </Field>
      <Field id="obs-ref" label="Reference range">
        <input id="obs-ref" class="field" bind:value={form.reference} maxlength="60" placeholder="3.9 - 6.1" />
      </Field>
      <Field id="obs-flag" label="Flag">
        <select id="obs-flag" class="field" bind:value={form.flag}>
          <option value="">None</option>
          <option value="H">H (high)</option>
          <option value="L">L (low)</option>
          <option value="*">* (abnormal)</option>
        </select>
      </Field>
    </div>
    <Field id="obs-metric" label="Catalog entry" hint="Links the value to its chart. Leave blank if none fits.">
      <select id="obs-metric" class="field" bind:value={form.metricCode}>
        <option value="">Not linked</option>
        {#each catalog.groups as g (g.category)}
          <optgroup label={g.label}>
            {#each g.metrics as m (m.code)}
              <option value={m.code}>{m.name}{m.unit ? ` (${m.unit})` : ''}</option>
            {/each}
          </optgroup>
        {/each}
      </select>
    </Field>
    <div class="grid grid-cols-2 gap-4">
      <Field id="obs-date" label="Sample date" hint="Blank uses the visit date.">
        <input id="obs-date" class="field" type="date" bind:value={form.sampleDate} />
      </Field>
      {#if pages.length > 0}
        <Field id="obs-page" label="Page">
          <select id="obs-page" class="field" bind:value={form.pageId}>
            <option value="">Not set</option>
            {#each pages as p, i (p.id)}<option value={p.id}>Page {i + 1}</option>{/each}
          </select>
        </Field>
      {/if}
    </div>
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="obs-form" disabled={save.busy}>
      {save.busy ? 'Saving…' : 'Save'}
    </button>
  {/snippet}
</Modal>
