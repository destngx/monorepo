<script lang="ts">
  import MaskedText from '$lib/components/MaskedText.svelte';
  import type { Insight } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { insightIcon, severityStatus } from './severity';
  import { AITask } from './task.svelte';

  interface Props {
    title: string;
    subtitle?: string;
    load: (force: boolean, signal: AbortSignal) => Promise<Insight>;
    /** Idle hint: where the figures the model reads come from. */
    hint?: string;
    class?: string;
  }

  let {
    title,
    subtitle,
    load,
    hint = 'AI review of the data on this page. Figures come from the ledger.',
    class: className = '',
  }: Props = $props();

  // Rebuilt when the loader changes (e.g. another month), dropping old results.
  const task = $derived(new AITask(load));
</script>

<Card {title} {subtitle} class={className}>
  {#snippet actions()}
    {#if task.status === 'done'}
      <button
        type="button"
        class="btn-ghost px-2"
        aria-label="Regenerate {title}"
        title="Regenerate"
        onclick={() => task.run(true)}
      >
        <Icon name="refresh" size={16} />
      </button>
    {/if}
  {/snippet}
  {#if task.status === 'done' && task.value}
    {@const insight = task.value}
    <p class="mb-3 font-medium"><MaskedText text={insight.summary} /></p>
    <ul class="space-y-2">
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
    {#if insight.actionItems.length}
      <h3 class="mt-4 mb-2 text-xs font-semibold tracking-wide text-slate-600 uppercase dark:text-slate-400">
        Next steps
      </h3>
      <ul class="space-y-1 text-sm">
        {#each insight.actionItems as item, index (index)}
          <li class="flex gap-2">
            <Icon name="check" size={16} class="mt-0.5 text-emerald-600" /><MaskedText text={item} />
          </li>
        {/each}
      </ul>
    {/if}
  {:else}
    <AIStatusLine {task} {hint} />
  {/if}
</Card>
