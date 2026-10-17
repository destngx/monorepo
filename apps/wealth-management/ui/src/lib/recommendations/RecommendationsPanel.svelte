<script lang="ts">
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import type { Insight, Recommendation, RecommendationList, RecommendationStatusInput } from '$lib/api/types';
  import AIStatusLine from '$lib/ai/AIStatusLine.svelte';
  import Markdown from '$lib/ai/Markdown.svelte';
  import { insightIcon, severityStatus } from '$lib/ai/severity';
  import { AITask } from '$lib/ai/task.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import RecommendationItem from './RecommendationItem.svelte';
  import { headline, recordNote } from './recommendations';

  // A page's open recommendations with the button that runs the page's
  // review (ROADMAP R7). The review's summary shows above the list; the
  // recommendations are what persists.
  interface Props {
    title: string;
    subtitle?: string;
    /** App route whose open recommendations are listed. */
    page: string;
    review: (force: boolean, signal: AbortSignal) => Promise<Insight>;
    /** Idle hint: what the review reads. */
    hint?: string;
    class?: string;
  }

  let {
    title,
    subtitle,
    page,
    review,
    hint = 'The review reads the figures on this page and adds what is worth doing to your recommendations.',
    class: className = '',
  }: Props = $props();

  // Rebuilt when the loader changes (e.g. another month), dropping old results.
  const task = $derived(new AITask(review));
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

  $effect(() => {
    void loadList();
  });

  async function runReview() {
    await task.run(true);
    if (task.status === 'done') await loadList(true);
  }

  async function setStatus(item: Recommendation, input: RecommendationStatusInput) {
    busy = true;
    try {
      await api.setRecommendationStatus(item.id, item.etag, input);
      await loadList(true);
    } catch (error) {
      toasts.error(error, 'Could not update the recommendation');
      await loadList(true);
    } finally {
      busy = false;
    }
  }
</script>

<Card {title} {subtitle} class={className}>
  {#snippet actions()}
    <a class="btn-ghost px-2 text-xs" href="{base}/recommendations?page={encodeURIComponent(page)}">All</a>
    <button type="button" class="btn-ghost px-2 text-xs" disabled={task.status === 'loading'} onclick={runReview}
      ><Icon name="sparkles" size={16} /> {task.status === 'idle' ? 'Review' : 'Review again'}</button
    >
  {/snippet}

  {#if task.status === 'loading' || task.status === 'error'}
    <div class="mb-3"><AIStatusLine {task} /></div>
  {:else if task.status === 'done' && task.value}
    {@const insight = task.value}
    <div class="mb-4">
      <p class="font-medium"><MaskedText text={insight.summary} /></p>
      {#if insight.recorded}
        <p class="muted mt-1 text-xs" role="status">{recordNote(insight.recorded)}</p>
      {/if}
      {#if insight.sections.length}
        <details class="mt-2 text-sm">
          <summary class="muted cursor-pointer text-xs">Full review</summary>
          <ul class="mt-2 space-y-2">
            {#each insight.sections as section, index (index)}
              {@const status = severityStatus(section.severity)}
              <li class="flex gap-3 rounded-xl border p-3 {status.box}">
                <Icon name={insightIcon[section.icon]} size={18} class="mt-0.5" />
                <div class="min-w-0 text-slate-800 dark:text-slate-200">
                  <p class="text-sm font-semibold">
                    <span class="sr-only">{status.label}:</span><MaskedText text={section.title} />
                  </p>
                  <Markdown text={section.content} class="mt-1" />
                </div>
              </li>
            {/each}
          </ul>
        </details>
      {/if}
    </div>
  {/if}

  {#if listError}
    <p class="text-sm text-rose-700 dark:text-rose-300" role="alert">{listError}</p>
  {:else if list === null}
    <div
      class="h-3 w-2/3 animate-pulse rounded bg-slate-900/10 dark:bg-white/10"
      role="status"
      aria-label="Loading"
    ></div>
  {:else if list.items.length === 0}
    <p class="muted text-sm">{task.status === 'idle' ? hint : 'No open recommendations for this page.'}</p>
  {:else}
    <p class="mb-2 text-sm font-medium"><MaskedText text={headline(list.summary)} /></p>
    <ul class="space-y-2">
      {#each list.items as item (item.id)}
        <RecommendationItem {item} {busy} onstatus={setStatus} />
      {/each}
    </ul>
  {/if}
</Card>
