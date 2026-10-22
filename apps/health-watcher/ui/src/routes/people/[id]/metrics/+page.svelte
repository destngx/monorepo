<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import { Button, EmptyState, Icon, PageTitle } from '@libs/ui-kit';
  import FlagTag from '$lib/components/FlagTag.svelte';
  import MeasurementsModal from '$lib/components/MeasurementsModal.svelte';
  import PersonTabs from '$lib/components/PersonTabs.svelte';
  import { formatDate, metricCategoryLabels } from '$lib/format';
  import { formatChange, pointSource, pointValue } from '$lib/metrics';
  import { session } from '$lib/state/session.svelte';

  // A person's metrics dashboard (HW-MET-4): the latest value of each metric
  // with the paper's flag and the change since the previous value.
  let { data } = $props();
  let adding = $state(false);

  const person = $derived(data.overview.person);
  const count = $derived(data.overview.groups.reduce((n, g) => n + g.items.length, 0));
  const flagged = $derived(data.overview.groups.reduce((n, g) => n + g.items.filter((i) => i.latest.flag).length, 0));
</script>

<svelte:head><title>Metrics · {person.displayName} · Health Watcher</title></svelte:head>

<PageTitle
  title={person.displayName}
  subtitle={count
    ? `${count} metrics · ${flagged ? `${flagged} flagged on their latest paper` : 'none flagged on their latest paper'}`
    : 'No metrics yet'}
>
  {#snippet actions()}
    {#if session.canEdit}
      <Button variant="primary" icon="plus" onclick={() => (adding = true)}>Add measurements</Button>
    {/if}
  {/snippet}
</PageTitle>

<PersonTabs personId={person.id} current="metrics" />

{#if data.overview.unlinked > 0}
  <p class="panel mb-5 flex items-start gap-2 p-3 text-sm" role="note">
    <Icon name="info" size={16} />
    <span
      >{data.overview.unlinked}
      {data.overview.unlinked === 1
        ? 'result is not linked to a metric, so it is'
        : 'results are not linked to a metric, so they are'}
      not charted. Link them from their document.</span
    >
  </p>
{/if}

{#each data.overview.groups as group (group.category)}
  <h2 class="eyebrow mt-6 mb-2">{metricCategoryLabels[group.category]}</h2>
  <ul class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
    {#each group.items as item (item.metric.code)}
      {@const v = pointValue(item.latest, item.metric)}
      <li>
        <a
          href="{base}/people/{person.id}/metrics/{item.metric.code}"
          class="panel interactive flex h-full flex-col gap-2 p-4"
        >
          <span class="flex items-start justify-between gap-2">
            <span class="min-w-0">
              <span class="block truncate font-medium">{item.metric.name}</span>
              <span class="muted block truncate text-xs">{item.metric.nameVi}</span>
            </span>
            <FlagTag flag={item.latest.flag} />
          </span>
          <span class="flex items-end justify-between gap-3">
            <span class="min-w-0">
              <span class="text-2xl font-semibold">{v.value}</span>
              <span class="muted text-sm">{v.unit}</span>
            </span>
            <Sparkline points={item.recent} />
          </span>
          <span class="muted mt-auto text-xs">
            {formatDate(item.latest.date)} · {pointSource(item.latest)}
            {#if item.change !== undefined && item.previous}
              <br /><span class="tabular">{formatChange(item.change)} {item.metric.unit}</span> since {formatDate(
                item.previous.date,
              )}
            {:else if item.count === 1}
              <br />First value
            {/if}
          </span>
        </a>
      </li>
    {/each}
  </ul>
{:else}
  <section class="panel">
    <EmptyState
      title="No metrics yet"
      detail="Approve a reading from the inbox, or add measurements such as weight or blood pressure."
    />
  </section>
{/each}

<MeasurementsModal
  open={adding}
  personId={person.id}
  onclose={() => (adding = false)}
  onsaved={() => {
    adding = false;
    void invalidateAll();
  }}
/>
