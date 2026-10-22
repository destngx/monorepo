<script lang="ts">
  import type { Metric } from '$lib/api/types';
  import { todayISO } from '$lib/format';
  import { typedVitals } from '$lib/metrics';
  import { action } from '$lib/state/action.svelte';
  import { catalog } from '$lib/state/catalog.svelte';
  import { session } from '$lib/state/session.svelte';
  import Field from './Field.svelte';
  import Modal from './Modal.svelte';

  // Vitals typed in for one day without a paper, such as a home blood
  // pressure reading (HW-MET-2). Empty fields are skipped.
  interface Props {
    open: boolean;
    personId: string;
    /** The metric to put first, when opened from its trend. */
    focus?: string;
    onclose: () => void;
    onsaved: () => void;
  }

  let { open, personId, focus, onclose, onsaved }: Props = $props();

  let date = $state('');
  let values = $state<Record<string, string>>({});
  let units = $state<Record<string, string>>({});

  const metrics = $derived.by(() => {
    const list = typedVitals.map((code) => catalog.byCode.get(code)).filter((m): m is Metric => !!m);
    return focus ? [...list.filter((m) => m.code === focus), ...list.filter((m) => m.code !== focus)] : list;
  });

  $effect(() => {
    if (!open) return;
    date = todayISO();
    values = {};
    units = {};
    void catalog.load();
  });

  const save = action(
    async () => {
      const filled = metrics.filter((m) => values[m.code]?.trim());
      if (!filled.length) throw new Error('Type at least one value.');
      await session.api.addMeasurements(personId, {
        date,
        values: filled.map((m) => ({ metricCode: m.code, value: values[m.code], unit: units[m.code] || m.unit })),
      });
      onsaved();
    },
    { errorContext: 'Could not save the measurements', success: 'Measurements saved.' },
  );
</script>

<Modal {open} title="Add measurements" {onclose} dismissible={!save.busy}>
  <form
    id="measurements-form"
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void save.run();
    }}
  >
    <p class="muted text-sm">For values taken without a paper, such as a home blood pressure reading.</p>
    <Field id="m-date" label="Date">
      <input id="m-date" class="field" type="date" bind:value={date} max={todayISO()} required />
    </Field>
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
      {#each metrics as m (m.code)}
        <Field id="m-{m.code}" label={m.name}>
          <div class="flex gap-2">
            <input
              id="m-{m.code}"
              class="field min-w-0 flex-1"
              inputmode="decimal"
              autocomplete="off"
              bind:value={values[m.code]}
              data-autofocus={m.code === metrics[0]?.code ? '' : undefined}
            />
            {#if m.conversions?.length}
              <select class="field w-24" aria-label="{m.name} unit" bind:value={units[m.code]}>
                <option value="">{m.unit}</option>
                {#each m.conversions as c (c.from)}<option value={c.from}>{c.from}</option>{/each}
              </select>
            {:else}
              <span class="muted self-center text-sm">{m.unit}</span>
            {/if}
          </div>
        </Field>
      {/each}
    </div>
    <p class="muted text-xs">BMI is computed from weight and the latest height.</p>
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="measurements-form" disabled={save.busy}
      >{save.busy ? 'Saving…' : 'Save measurements'}</button
    >
  {/snippet}
</Modal>
