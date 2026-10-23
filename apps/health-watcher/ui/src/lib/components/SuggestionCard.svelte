<script lang="ts">
  import { untrack } from 'svelte';
  import type { MetricCategory, MetricSuggestion } from '$lib/api/types';
  import { Button, Card, Field } from '@libs/ui-kit';
  import { formatDate, metricCategoryLabels } from '$lib/format';
  import { catalog } from '$lib/state/catalog.svelte';
  import { session } from '$lib/state/session.svelte';

  // One metric suggestion (HW-MET-8): the printed name, what the AI thinks
  // it is and why. A new metric's name, unit and category can be changed
  // before approving it.
  interface Props {
    suggestion: MetricSuggestion;
    ondecided: (added: boolean) => unknown;
  }

  let { suggestion, ondecided }: Props = $props();

  const categories = (Object.keys(metricCategoryLabels) as MetricCategory[]).filter((c) => c !== 'printed');
  const initial = untrack(() => suggestion.proposal);
  let name = $state(initial?.name ?? '');
  let nameVi = $state(initial?.nameVi ?? '');
  let unit = $state(initial?.unit ?? '');
  let category = $state<MetricCategory>(initial?.category ?? 'enzymes');

  const target = $derived(suggestion.metricCode ? catalog.byCode.get(suggestion.metricCode) : undefined);
  const id = $derived(`s-${suggestion.id}`);

  async function approve() {
    if (suggestion.kind === 'new') {
      await session.api.approveSuggestion(suggestion.id, { name, nameVi, unit, category });
    } else {
      await session.api.approveSuggestion(suggestion.id);
    }
    await ondecided(suggestion.kind === 'new');
  }

  async function reject() {
    await session.api.rejectSuggestion(suggestion.id);
    await ondecided(false);
  }
</script>

<Card>
  <div class="flex flex-wrap items-baseline justify-between gap-2">
    <h2 class="font-medium">
      {suggestion.analyte}{#if suggestion.unit}<span class="muted text-sm font-normal"> · {suggestion.unit}</span>{/if}
    </h2>
    <span class="muted text-xs">{suggestion.values} {suggestion.values === 1 ? 'value' : 'values'}</span>
  </div>

  {#if suggestion.kind === 'link'}
    <p class="mt-2 text-sm">
      Same test as <strong>{target?.name ?? suggestion.metricCode}</strong>{#if target}<span class="muted">
          · {target.nameVi}{target.unit ? ` · ${target.unit}` : ''}</span
        >{/if}
    </p>
  {:else if suggestion.proposal}
    {@const p = suggestion.proposal}
    <p class="mt-2 text-sm">A test the catalog doesn't have. Approving adds it as a new metric:</p>
    <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
      <Field id="{id}-name" label="Name">
        <input id="{id}-name" class="field" bind:value={name} maxlength="80" required disabled={!session.canEdit} />
      </Field>
      <Field id="{id}-vi" label="Vietnamese name">
        <input id="{id}-vi" class="field" bind:value={nameVi} maxlength="80" disabled={!session.canEdit} />
      </Field>
      <Field id="{id}-unit" label="Unit" hint="Empty for a result read as text.">
        <input id="{id}-unit" class="field" bind:value={unit} maxlength="30" disabled={!session.canEdit} />
      </Field>
      <Field id="{id}-cat" label="Category">
        <select id="{id}-cat" class="field" bind:value={category} disabled={!session.canEdit}>
          {#each categories as c (c)}<option value={c}>{metricCategoryLabels[c]}</option>{/each}
        </select>
      </Field>
    </div>
    <p class="muted mt-3 text-xs">
      {#if p.loinc}
        LOINC <a class="link" href="https://loinc.org/{p.loinc}/" target="_blank" rel="noopener noreferrer">{p.loinc}</a
        >{#if p.loincName}: {p.loincName}{/if}, checked with the U.S. National Library of Medicine.
      {:else}
        No LOINC code: the AI wasn't sure of one, or the National Library of Medicine didn't know it.
      {/if}
    </p>
    {#if p.about}
      <figure class="mt-3">
        <blockquote class="border-l-2 border-ink-900/15 pl-3 text-sm dark:border-ink-50/15">
          <p>{p.about.quote}</p>
        </blockquote>
        <figcaption class="muted mt-1 text-xs">
          <a class="link" href={p.about.url} target="_blank" rel="noopener noreferrer"
            >{p.about.source}: {p.about.title}</a
          >, as on {formatDate(p.about.fetched)}.
        </figcaption>
      </figure>
    {/if}
  {/if}

  <p class="muted mt-3 text-xs">Why: {suggestion.reason}</p>

  {#if session.canEdit}
    <div class="mt-4 flex flex-wrap gap-2">
      <Button
        variant="primary"
        icon="check"
        onclick={approve}
        busyText="Saving…"
        success={suggestion.kind === 'new' ? 'Metric added.' : 'Linked.'}
        errorContext="Could not approve the suggestion">{suggestion.kind === 'new' ? 'Add metric' : 'Link'}</Button
      >
      <Button icon="x" onclick={reject} errorContext="Could not reject the suggestion">Keep as printed</Button>
    </div>
  {/if}
</Card>
