<script lang="ts">
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { Recommendation, RecommendationStatusInput } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import RecommendationItem from '$lib/recommendations/RecommendationItem.svelte';
  import { GROUPS, groupByStatus, headline, STATUS_TONE } from '$lib/recommendations/recommendations';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import type { StatusTone } from '$lib/status';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();
  const list = $derived(data.list);
  const groups = $derived(groupByStatus(list.items));
  const tone = $derived<StatusTone>(list.summary.open > 0 ? 'warning' : 'good');
  let busy = $state(false);

  async function setStatus(item: Recommendation, input: RecommendationStatusInput) {
    busy = true;
    try {
      await api.setRecommendationStatus(item.id, item.etag, input);
    } catch (error) {
      toasts.error(error, 'Could not update the recommendation');
    } finally {
      await invalidateAll();
      busy = false;
    }
  }
</script>

<PageHeader
  title="Recommendations"
  subtitle="What the reviews and the engine checks suggest, until you act on it. Mark items done, dismiss or snooze them."
/>

<PageHeadline question="What should I do next?" {tone} id="recommendations-headline" class="mb-4">
  {#snippet answer()}<span><MaskedText text={headline(list.summary)} /></span>{/snippet}
  <p class="muted">
    {list.summary.snoozed} snoozed, {list.summary.done} done, {list.summary.dismissed} dismissed.
    {#if list.summary.openWithoutImpact}
      {list.summary.openWithoutImpact} open {list.summary.openWithoutImpact === 1 ? 'item has' : 'items have'} no estimate.
    {/if}
  </p>
  {#if data.page}
    <p class="mt-2">
      Showing {data.page}.
      <a class="text-indigo-600 hover:underline dark:text-indigo-400" href="{base}/recommendations">Show every page</a>
    </p>
  {/if}
  {#if !data.refreshed}
    <p class="mt-2 text-amber-700 dark:text-amber-300" role="status">
      The engine checks did not run: they write to the sheet, and this connection has no write access. The list below is
      as stored.
    </p>
  {/if}
</PageHeadline>

{#if list.issues.length}
  <Card title="Rows left out" subtitle="Rows of the Recommendations tab the engine could not read" class="mb-4">
    <ul class="space-y-1 text-sm">
      {#each list.issues as issue (issue.row)}
        <li>Row {issue.row}: {issue.message}</li>
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
