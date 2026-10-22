<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { TrendPoint } from '$lib/api/types';
  import { FLAGGED, SERIES } from '$lib/charts/palette';
  import TrendChart from '$lib/charts/TrendChart.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import FlagTag from '$lib/components/FlagTag.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MeasurementsModal from '$lib/components/MeasurementsModal.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import { formatDate } from '$lib/format';
  import { pointRange, pointSource, pointValue, typedVitals } from '$lib/metrics';
  import { session } from '$lib/state/session.svelte';

  // One metric of a person over time (HW-MET-3), with every value in a table
  // below the chart, newest first, each linked to its paper.
  let { data } = $props();
  let adding = $state(false);

  const person = $derived(data.trend.person);
  const metric = $derived(data.trend.metric);
  const rows = $derived([...data.trend.points].reverse());
  const typable = $derived(typedVitals.includes(metric.code));
  const hasComputed = $derived(data.trend.points.some((p) => p.source === 'computed'));
  const flags = $derived(new Set(data.trend.points.filter((p) => p.value !== undefined && p.flag).map((p) => p.flag)));

  function open(p: TrendPoint) {
    if (p.documentId) void goto(`${base}/documents/${p.documentId}`);
  }

  async function remove(p: TrendPoint) {
    if (!p.observationId || !confirm(`Delete the value typed in for ${formatDate(p.date)}?`)) return;
    await session.api.deleteObservation(p.observationId);
    await invalidateAll();
  }
</script>

<svelte:head><title>{metric.name} · {person.displayName} · Health Watcher</title></svelte:head>

<a class="link mb-3 inline-flex items-center gap-1 text-sm" href="{base}/people/{person.id}/metrics"
  ><Icon name="chevronLeft" size={14} />{person.displayName}'s metrics</a
>

<PageTitle
  title={metric.name}
  subtitle="{metric.nameVi} · {data.trend.points.length} values{metric.unit ? ` · in ${metric.unit}` : ''}"
>
  {#snippet actions()}
    {#if session.canEdit && typable}
      <Button variant="primary" icon="plus" onclick={() => (adding = true)}>Add a measurement</Button>
    {/if}
  {/snippet}
</PageTitle>

<Card class="mb-5">
  <TrendChart points={data.trend.points} {metric} onopen={open} />
  <ul class="muted mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs" aria-label="Chart key">
    <li class="flex items-center gap-1.5">
      <svg width="12" height="12" aria-hidden="true"><circle cx="6" cy="6" r="4" fill={SERIES} /></svg>Value
    </li>
    <li class="flex items-center gap-1.5">
      <span class="inline-block size-3 bg-ink-500/12 dark:bg-ink-300/12" aria-hidden="true"></span>Range printed on the
      paper
    </li>
    {#if flags.has('H')}
      <li class="flex items-center gap-1.5">
        <svg width="12" height="12" aria-hidden="true"><path d="M6,1 L11,10 L1,10 Z" fill={FLAGGED} /></svg>Flagged high
        on the paper
      </li>
    {/if}
    {#if flags.has('L')}
      <li class="flex items-center gap-1.5">
        <svg width="12" height="12" aria-hidden="true"><path d="M6,11 L11,2 L1,2 Z" fill={FLAGGED} /></svg>Flagged low
        on the paper
      </li>
    {/if}
    {#if flags.has('*')}
      <li class="flex items-center gap-1.5">
        <svg width="12" height="12" aria-hidden="true"><path d="M6,1 L11,6 L6,11 L1,6 Z" fill={FLAGGED} /></svg>Flagged
        abnormal on the paper
      </li>
    {/if}
    {#if hasComputed}
      <li class="flex items-center gap-1.5">
        <svg width="12" height="12" aria-hidden="true"
          ><circle cx="6" cy="6" r="4" fill="var(--surface)" stroke={SERIES} stroke-width="2" /></svg
        >Computed from weight and the latest height
      </li>
    {/if}
  </ul>
  <p class="muted mt-2 text-xs">
    Ranges and flags are the ones printed on each paper; the app adds none of its own. Labs differ, so compare with a
    doctor.
  </p>
</Card>

<Card title="All values">
  <div class="relative overflow-x-auto">
    <table class="w-full text-sm">
      <thead class="muted text-left text-xs">
        <tr>
          <th class="py-2 pr-4 font-medium">Date</th>
          <th class="py-2 pr-4 font-medium">Value</th>
          <th class="py-2 pr-4 font-medium">Paper's range</th>
          <th class="py-2 pr-4 font-medium">Flag</th>
          <th class="py-2 pr-4 font-medium">From</th>
          <th class="py-2"><span class="sr-only">Actions</span></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
        {#each rows as p, i (p.observationId ?? `${p.source}-${p.date}-${i}`)}
          {@const v = pointValue(p, metric)}
          <tr>
            <td class="py-2 pr-4 whitespace-nowrap tabular">{formatDate(p.date)}</td>
            <td class="py-2 pr-4 whitespace-nowrap">
              <span class="font-medium tabular">{v.value}</span> <span class="muted">{v.unit}</span>
              {#if p.value !== undefined && p.printedUnit && p.printedUnit !== metric.unit}
                <span class="muted block text-xs">printed {p.printed} {p.printedUnit}</span>
              {:else if p.value === undefined && metric.unit && p.printedUnit !== undefined && p.printedUnit !== ''}
                <span class="muted block text-xs">not charted: no conversion from {p.printedUnit}</span>
              {/if}
            </td>
            <td class="py-2 pr-4 whitespace-nowrap tabular">{pointRange(p) || '-'}</td>
            <td class="py-2 pr-4"><FlagTag flag={p.flag} /></td>
            <td class="py-2 pr-4 whitespace-nowrap">
              {#if p.documentId}
                <a class="link" href="{base}/documents/{p.documentId}">{pointSource(p)}</a>
              {:else}
                <span class="muted">{pointSource(p)}</span>
              {/if}
            </td>
            <td class="py-2 text-right">
              {#if session.canEdit && p.source === 'typed'}
                <Button size="icon" icon="trash" label="Delete this value" onclick={() => remove(p)} />
              {/if}
            </td>
          </tr>
        {:else}
          <tr><td colspan="6" class="muted py-6 text-center">No values yet.</td></tr>
        {/each}
      </tbody>
    </table>
  </div>
</Card>

<MeasurementsModal
  open={adding}
  personId={person.id}
  focus={metric.code}
  onclose={() => (adding = false)}
  onsaved={() => {
    adding = false;
    void invalidateAll();
  }}
/>
