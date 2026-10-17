<script lang="ts">
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import type { Recommendation, RecommendationList, RecommendationStatusInput } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import LoadError from '$lib/components/LoadError.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonHeadline from '$lib/components/skeleton/SkeletonHeadline.svelte';
  import Spinner from '$lib/components/Spinner.svelte';
  import RecommendationItem from '$lib/recommendations/RecommendationItem.svelte';
  import {
    checksSkipped,
    GROUPS,
    groupByStatus,
    headline,
    STATUS_TONE,
    storedOrChecked,
  } from '$lib/recommendations/recommendations';
  import type { StatusTone } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';

  // The header shows at once, the stored list as soon as it is read, and the
  // rule refresh streams in behind a "Checking the rules" status: its list
  // replaces the stored one when it lands.
  let { data } = $props();
  const checks = track(() => data.refresh);
  const listData = $derived(storedOrChecked(data.list, data.refresh));

  // The newest list wins: the stored read until the refresh lands, then the
  // refresh, then a re-read after each status change. A refresh that lands
  // after a change is re-read too, so it never brings back the old status.
  let current = $state<RecommendationList | null>(null);
  let changedSinceLoad = false;
  $effect(() => {
    const stored = data.list;
    const refreshed = data.refresh;
    let live = true;
    let checked = false;
    changedSinceLoad = false;
    void refreshed.then((result) => {
      if (!live) return;
      checked = true;
      if (!result.ok) return; // shown by the checks status below
      if (changedSinceLoad) void reread();
      else current = result.value;
    });
    void stored.then((result) => {
      if (live && !checked && !changedSinceLoad && result.ok) current = result.value;
    });
    return () => {
      live = false;
    };
  });

  async function reread() {
    try {
      current = await api.recommendations({ page: data.page }, { force: true });
    } catch (error) {
      toasts.error(error, 'Could not reload the recommendations');
    }
  }

  const toneOf = (list: RecommendationList): StatusTone => (list.summary.open > 0 ? 'warning' : 'good');

  let busy = $state(false);
  // The item's button shows the busy state and toasts the result; the list is
  // re-read either way, so a failed change shows the stored state.
  async function setStatus(item: Recommendation, input: RecommendationStatusInput) {
    busy = true;
    changedSinceLoad = true;
    try {
      await api.setRecommendationStatus(item.id, item.etag, input);
    } finally {
      await reread();
      busy = false;
    }
  }
</script>

<PageHeader
  title="Recommendations"
  subtitle="What the reviews and the engine checks suggest, until you act on it. Mark items done, dismiss or snooze them."
>
  {#snippet actions()}
    {#if checks.pending}
      <span class="muted inline-flex items-center gap-1.5 text-sm" role="status" data-testid="checking-rules">
        <Spinner size={14} /> Checking the rules…
      </span>
    {/if}
  {/snippet}
</PageHeader>

{#if data.page}
  <p class="muted mb-3 text-sm">
    Showing {data.page}. <a class="link" href="{base}/recommendations">Show every page</a>
  </p>
{/if}

{#if checksSkipped(checks.failure)}
  <p class="mb-4 text-sm text-amber-700 dark:text-amber-300" role="status">
    The engine checks did not run: they write to the sheet, and this connection has no write access. The list below is
    as stored.
  </p>
{:else if checks.failure}
  <LoadError failure={checks.failure} label="the engine checks" retry={() => invalidateAll()} compact class="mb-4" />
{/if}

<Deferred data={listData} label="recommendations">
  {#snippet children(loaded)}
    {@const list = current ?? loaded}
    {@const groups = groupByStatus(list.items)}
    <div>
      <PageHeadline question="What should I do next?" tone={toneOf(list)} id="recommendations-headline" class="mb-4">
        {#snippet answer()}<span><MaskedText text={headline(list.summary)} /></span>{/snippet}
        <p class="muted">
          {list.summary.snoozed} snoozed, {list.summary.done} done, {list.summary.dismissed} dismissed.
          {#if list.summary.openWithoutImpact}
            {list.summary.openWithoutImpact} open {list.summary.openWithoutImpact === 1 ? 'item has' : 'items have'} no estimate.
          {/if}
        </p>
      </PageHeadline>

      {#if list.issues.length}
        <Card title="Rows left out" subtitle="Rows of the Recommendations tab the engine could not read" class="mb-4">
          <ul class="space-y-1 text-sm">
            {#each list.issues as issue (issue.row)}
              <li>Row {issue.row}: <MaskedText text={issue.message} /></li>
            {/each}
          </ul>
        </Card>
      {/if}

      {#if list.items.length === 0}
        <EmptyState
          title="No recommendations yet"
          detail="Run a review on Accounts, Budget, Transactions, Cash Flow, Credit Cards or Health, or the briefing on the dashboard. Engine checks add items here on their own."
        />
      {:else}
        <div class="space-y-4">
          {#each GROUPS as group (group.status)}
            {#if groups[group.status].length}
              <Card>
                <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold">
                  <Badge status={STATUS_TONE[group.status]}>{group.title}</Badge>
                  <span class="muted text-xs">{groups[group.status].length}</span>
                </h2>
                <ul class="space-y-2">
                  {#each groups[group.status] as item (item.id)}
                    <RecommendationItem {item} {busy} onstatus={setStatus} />
                  {/each}
                </ul>
              </Card>
            {/if}
          {/each}
        </div>
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonHeadline question="What should I do next?" label="recommendations" />
      <SkeletonCard rows={4} lines={0} label="the list" class="mt-4" />
    </div>
  {/snippet}
</Deferred>
