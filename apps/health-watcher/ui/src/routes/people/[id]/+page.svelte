<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import Button from '$lib/components/Button.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import VisitFormModal from '$lib/components/VisitFormModal.svelte';
  import { ageOn, formatDate, options, todayISO, visitTypeLabels } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  let adding = $state(false);
  const age = $derived(ageOn(data.person.birthDate, todayISO()));
  const filtered = $derived(Object.values(data.filter).some(Boolean));

  // Filters live in the URL, so a filtered timeline can be bookmarked (HW-FIND-1).
  function setFilter(key: string, value: string) {
    const url = new URL(page.url);
    if (value) url.searchParams.set(key, value);
    else url.searchParams.delete(key);
    void goto(url, { keepFocus: true, noScroll: true, replaceState: true });
  }

  // Visits grouped by year, newest first.
  const years = $derived.by(() => {
    const groups: { year: string; visits: typeof data.visits }[] = [];
    for (const v of data.visits) {
      const year = v.date.slice(0, 4);
      const last = groups.at(-1);
      if (last?.year === year) last.visits.push(v);
      else groups.push({ year, visits: [v] });
    }
    return groups;
  });
</script>

<PageTitle
  title={data.person.displayName}
  subtitle="{age !== null ? `${age} years · ` : ''}{data.person.visitCount} visits in total"
>
  {#snippet actions()}
    {#if session.canEdit}
      <a class="btn-ghost" href="{base}/inbox"><Icon name="upload" size={16} />Upload</a>
      <Button variant="primary" icon="plus" onclick={() => (adding = true)}>Add a visit</Button>
    {/if}
  {/snippet}
</PageTitle>

<section class="glass mb-5 grid gap-3 p-4 sm:grid-cols-4">
  <div>
    <label class="label" for="f-type">Type</label>
    <select
      id="f-type"
      class="field"
      value={data.filter.type}
      onchange={(e) => setFilter('type', e.currentTarget.value)}
    >
      <option value="">All types</option>
      {#each options(visitTypeLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
    </select>
  </div>
  <div>
    <label class="label" for="f-provider">Place</label>
    <select
      id="f-provider"
      class="field"
      value={data.filter.providerId}
      onchange={(e) => setFilter('providerId', e.currentTarget.value)}
    >
      <option value="">All places</option>
      {#each data.providers as p (p.id)}<option value={p.id}>{p.name}</option>{/each}
    </select>
  </div>
  <div>
    <label class="label" for="f-from">From</label>
    <input
      id="f-from"
      class="field"
      type="date"
      value={data.filter.from}
      onchange={(e) => setFilter('from', e.currentTarget.value)}
    />
  </div>
  <div>
    <label class="label" for="f-to">To</label>
    <input
      id="f-to"
      class="field"
      type="date"
      value={data.filter.to}
      onchange={(e) => setFilter('to', e.currentTarget.value)}
    />
  </div>
</section>

{#each years as group (group.year)}
  <h2 class="muted mt-6 mb-2 text-sm font-semibold tracking-wide">{group.year}</h2>
  <ol class="glass divide-y divide-slate-900/5 dark:divide-white/5">
    {#each group.visits as v (v.id)}
      <li>
        <a href="{base}/visits/{v.id}" class="row-link flex items-center gap-4 px-4 py-3">
          <span class="w-20 shrink-0 text-sm tabular">{formatDate(v.date)}</span>
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium">{v.title || visitTypeLabels[v.type]}</span>
            <span class="muted block truncate text-xs">
              {visitTypeLabels[v.type]}{v.provider ? ` · ${v.provider.name}` : ''}
            </span>
          </span>
          {#if v.flaggedCount > 0}
            <span
              class="flex shrink-0 items-center gap-1 rounded-full bg-rose-500/10 px-2 py-0.5 text-xs text-rose-700 dark:text-rose-300"
              title="{v.flaggedCount} values flagged high or low"><Icon name="alert" size={12} />{v.flaggedCount}</span
            >
          {/if}
          <span class="muted flex shrink-0 items-center gap-1 text-xs"
            ><Icon name="file" size={14} />{v.documentCount}</span
          >
        </a>
      </li>
    {/each}
  </ol>
{:else}
  <section class="glass">
    <EmptyState
      title={filtered ? 'No visits match these filters' : 'No visits yet'}
      detail={filtered ? 'Clear a filter to see more.' : 'Upload a result in the inbox, or add a visit by hand.'}
    />
  </section>
{/each}

<VisitFormModal
  open={adding}
  visit={null}
  personId={data.person.id}
  people={data.people}
  providers={data.providers}
  onclose={() => (adding = false)}
  onsaved={(v) => {
    adding = false;
    void goto(`${base}/visits/${v.id}`);
  }}
/>
