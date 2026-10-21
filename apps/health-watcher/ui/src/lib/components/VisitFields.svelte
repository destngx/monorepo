<script lang="ts">
  import type { Person, Provider, VisitInput } from '$lib/api/types';
  import { options, providerKindLabels, todayISO, visitTypeLabels } from '$lib/format';
  import Field from './Field.svelte';

  // The fields of a visit, shared by the visit form and inbox filing.
  interface Props {
    value: VisitInput;
    people: Person[];
    providers: Provider[];
    /** Prefix for element ids, so two forms on a page don't clash. */
    idPrefix?: string;
  }

  let { value = $bindable(), people, providers, idPrefix = 'v' }: Props = $props();
  const known = $derived(providers.some((p) => p.name.toLowerCase() === value.providerName.trim().toLowerCase()));
</script>

<div class="space-y-4">
  <div class="grid gap-4 sm:grid-cols-2">
    <Field id="{idPrefix}-person" label="Person">
      <select id="{idPrefix}-person" class="field" bind:value={value.personId} required>
        <option value="" disabled>Choose…</option>
        {#each people as p (p.id)}<option value={p.id}>{p.displayName}</option>{/each}
      </select>
    </Field>
    <Field id="{idPrefix}-date" label="Date">
      <input id="{idPrefix}-date" class="field" type="date" bind:value={value.date} max={todayISO()} required />
    </Field>
  </div>
  <div class="grid gap-4 sm:grid-cols-2">
    <Field id="{idPrefix}-type" label="Type">
      <select id="{idPrefix}-type" class="field" bind:value={value.type}>
        {#each options(visitTypeLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
      </select>
    </Field>
    <Field id="{idPrefix}-title" label="Title" hint="Optional, e.g. Yearly check-up.">
      <input id="{idPrefix}-title" class="field" bind:value={value.title} maxlength="120" />
    </Field>
  </div>
  <div class="grid gap-4 sm:grid-cols-[2fr_1fr]">
    <Field id="{idPrefix}-provider" label="Hospital, clinic or lab" hint="Names match ignoring accents and case.">
      <input
        id="{idPrefix}-provider"
        class="field"
        list="{idPrefix}-providers"
        bind:value={value.providerName}
        maxlength="120"
      />
      <datalist id="{idPrefix}-providers">
        {#each providers as p (p.id)}<option value={p.name}></option>{/each}
      </datalist>
    </Field>
    {#if value.providerName.trim() && !known}
      <Field id="{idPrefix}-pkind" label="New place is a">
        <select id="{idPrefix}-pkind" class="field" bind:value={value.providerKind}>
          {#each options(providerKindLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
      </Field>
    {/if}
  </div>
  <Field id="{idPrefix}-note" label="Note">
    <textarea id="{idPrefix}-note" class="field" rows="2" bind:value={value.note} maxlength="4000"></textarea>
  </Field>
</div>
