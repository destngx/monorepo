<script lang="ts">
  import type { FindingInput, Page } from '$lib/api/types';
  import { findingKindLabels, options } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import Field from './Field.svelte';
  import Modal from './Modal.svelte';

  // Adds or edits one finding: a diagnosis, prescription line, procedure or note.
  interface Props {
    value: FindingInput | null;
    title: string;
    pages: Page[];
    onsave: (input: FindingInput) => Promise<unknown>;
    onclose: () => void;
  }

  let { value, title, pages, onsave, onclose }: Props = $props();
  let form = $state<FindingInput>({
    kind: 'diagnosis',
    text: '',
    icd10: '',
    drug: '',
    dose: '',
    frequency: '',
    duration: '',
    pageId: '',
  });

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
    id="finding-form"
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void save.run();
    }}
  >
    <div class="grid grid-cols-2 gap-4">
      <Field id="fd-kind" label="Kind">
        <select id="fd-kind" class="field" bind:value={form.kind}>
          {#each options(findingKindLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
      {#if form.kind === 'diagnosis'}
        <Field id="fd-icd" label="ICD-10 code" hint="Only when printed, e.g. J02.9.">
          <input id="fd-icd" class="field uppercase" bind:value={form.icd10} maxlength="8" />
        </Field>
      {/if}
    </div>
    <Field id="fd-text" label="Text">
      <textarea id="fd-text" class="field min-h-20" bind:value={form.text} required maxlength="2000"></textarea>
    </Field>
    {#if form.kind === 'prescription'}
      <div class="grid grid-cols-2 gap-4">
        <Field id="fd-drug" label="Medicine">
          <input id="fd-drug" class="field" bind:value={form.drug} maxlength="120" />
        </Field>
        <Field id="fd-dose" label="Dose">
          <input id="fd-dose" class="field" bind:value={form.dose} maxlength="60" />
        </Field>
        <Field id="fd-freq" label="How often">
          <input id="fd-freq" class="field" bind:value={form.frequency} maxlength="60" />
        </Field>
        <Field id="fd-dur" label="For how long">
          <input id="fd-dur" class="field" bind:value={form.duration} maxlength="60" />
        </Field>
      </div>
    {/if}
    {#if pages.length > 0}
      <Field id="fd-page" label="Page">
        <select id="fd-page" class="field" bind:value={form.pageId}>
          <option value="">Not set</option>
          {#each pages as p, i (p.id)}<option value={p.id}>Page {i + 1}</option>{/each}
        </select>
      </Field>
    {/if}
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="finding-form" disabled={save.busy}>
      {save.busy ? 'Saving…' : 'Save'}
    </button>
  {/snippet}
</Modal>
