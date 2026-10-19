<script lang="ts">
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { base } from '$app/paths';
  import type { Insight } from '$lib/api/types';
  import AIStatusLine from '$lib/ai/AIStatusLine.svelte';
  import Markdown from '$lib/ai/Markdown.svelte';
  import { insightIcon, severityStatus } from '$lib/ai/severity';
  import { AITask } from '$lib/ai/task.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import OpenRecommendations from './OpenRecommendations.svelte';
  import { recordNote } from './recommendations';

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
  // Bumped after a review so the list re-reads what it filed.
  let reviewed = $state(0);

  async function runReview() {
    await task.run(true);
    if (task.status === 'done') reviewed++;
  }
</script>

<Card {title} {subtitle} class={className}>
  {#snippet actions()}
    <a
      class="btn-ghost px-2 text-xs"
      href="{base}/recommendations?page={encodeURIComponent(page)}"
      title="All recommendations for this page">All</a
    >
    <!-- The review reports its own progress and errors below (AIStatusLine). -->
    <Button
      size="sm"
      icon="sparkles"
      busy={task.status === 'loading'}
      busyText="Reviewing…"
      onclick={() => void runReview()}>{task.status === 'idle' ? 'Review' : 'Review again'}</Button
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

  <OpenRecommendations
    {page}
    refresh={reviewed}
    empty={task.status === 'idle' ? hint : 'No open recommendations for this page.'}
  />
</Card>
