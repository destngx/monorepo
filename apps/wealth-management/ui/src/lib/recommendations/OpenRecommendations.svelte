<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Recommendation, RecommendationList, RecommendationStatusInput } from '$lib/api/types';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import RecommendationItem from './RecommendationItem.svelte';
  import { headline } from './recommendations';

  // A page's open recommendations with Done / Dismiss / Snooze (WM-REC-6).
  // Shared by the review panels and the budget advisor (WM-BUD-11).
  interface Props {
    /** App route whose open recommendations are listed. */
    page: string;
    /** Shown when nothing is open. */
    empty: string;
    /** Bump to re-read the list (bypassing the cache) after items were filed. */
    refresh?: number;
  }

  let { page, empty, refresh = 0 }: Props = $props();

  let list = $state<RecommendationList | null>(null);
  let listError = $state('');
  let busy = $state(false);

  async function loadList(force = false) {
    try {
      list = await api.recommendations({ page, status: 'open' }, { force });
      listError = '';
    } catch (error) {
      listError = error instanceof Error ? error.message : String(error);
    }
  }

  // Runs again when `page` or `refresh` changes; a bump bypasses the cache.
  $effect(() => {
    void loadList(refresh > 0);
  });

  // The item's button shows the busy state and toasts the result; the list is
  // re-read either way, so a failed change shows the stored state.
  async function setStatus(item: Recommendation, input: RecommendationStatusInput) {
    busy = true;
    try {
      await api.setRecommendationStatus(item.id, item.etag, input);
    } finally {
      await loadList(true);
      busy = false;
    }
  }
</script>

{#if listError}
  <p class="text-sm text-rose-700 dark:text-rose-300" role="alert">{listError}</p>
{:else if list === null}
  <div
    class="h-3 w-2/3 animate-pulse rounded bg-stone-900/10 dark:bg-white/10"
    role="status"
    aria-label="Loading"
  ></div>
{:else if list.items.length === 0}
  <p class="muted text-sm">{empty}</p>
{:else}
  <p class="mb-2 text-sm font-medium"><MaskedText text={headline(list.summary)} /></p>
  <ul class="space-y-2">
    {#each list.items as item (item.id)}
      <RecommendationItem {item} {busy} onstatus={setStatus} />
    {/each}
  </ul>
{/if}
