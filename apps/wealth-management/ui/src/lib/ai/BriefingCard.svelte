<script lang="ts">
  import { onMount } from 'svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import { Card, Icon } from '@libs/ui-kit';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { severityStatus } from './severity';
  import { AITask } from './task.svelte';
  import { recordNote } from '$lib/recommendations/recommendations';

  // Dashboard intelligence briefing (WM-DASH-6, WM-AI-6). Compact (the home
  // page, ROADMAP R5) keeps the alerts and the briefing, and folds the
  // pattern cards away. It loads on open: the engine reuses a briefing of
  // the same facts for 30 minutes, so only the refresh button asks the AI
  // again. `after` is the page data to load first (WM-AI-16).
  let { compact = false, after }: { compact?: boolean; after?: Promise<unknown> } = $props();
  const task = new AITask((force, signal) => api.briefing({ force, signal }));

  onMount(() => {
    void task.run(false, { after });
    return () => task.cancel();
  });

  const patternLabels: Record<string, string> = {
    'spending-forecast': 'Spending forecast',
    'portfolio-health': 'Portfolio health',
    'what-if': 'What if',
    anomaly: 'Anomaly',
    benchmark: 'Benchmark',
    'investment-strategy': 'Investment strategy',
  };
</script>

<Card title="AI briefing" subtitle="Executive summary of this month">
  {#snippet actions()}
    {#if task.status === 'done'}
      <button type="button" class="btn-ghost px-2" aria-label="Regenerate briefing" onclick={() => task.run(true)}
        ><Icon name="refresh" size={16} /></button
      >
    {/if}
  {/snippet}
  {#if task.status === 'done' && task.value}
    {@const briefing = task.value}
    {#if briefing.alerts.length}
      <ul class="mb-4 grid grid-cols-1 gap-2 md:grid-cols-2">
        {#each briefing.alerts as alert, index (index)}
          {@const status = severityStatus(alert.severity)}
          <li class="rounded-xl border p-3 text-sm {status.box}">
            <p class="flex items-start gap-1.5 font-semibold">
              <Icon name={status.icon} size={16} class="mt-0.5" /><span class="sr-only">{status.label}:</span><span
                ><MaskedText text={alert.title} /></span
              >
            </p>
            <p class="text-stone-700 dark:text-stone-300"><MaskedText text={alert.message} /></p>
          </li>
        {/each}
      </ul>
    {/if}
    <Markdown text={briefing.briefing} />
    {#if briefing.recorded}
      <p class="muted mt-3 text-xs" role="status">
        {recordNote(briefing.recorded)}
        <a class="text-accent-600 hover:underline dark:text-accent-400" href="{base}/recommendations"
          >See recommendations</a
        >
      </p>
    {/if}
    {#if briefing.patterns.length}
      {#snippet patterns()}
        <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {#each briefing.patterns as pattern, index (index)}
            <div class="rounded-xl border border-stone-900/10 p-3 dark:border-white/10">
              <p class="muted text-xs font-medium tracking-wide uppercase">
                {patternLabels[pattern.kind] ?? pattern.kind}
              </p>
              <p class="mt-1 text-sm font-semibold"><MaskedText text={pattern.title} /></p>
              <Markdown text={pattern.content} class="mt-1" />
            </div>
          {/each}
        </div>
      {/snippet}
      {#if compact}
        <details class="mt-3 text-sm">
          <summary class="cursor-pointer text-xs font-medium">Show {briefing.patterns.length} patterns</summary>
          {@render patterns()}
        </details>
      {:else}
        {@render patterns()}
      {/if}
    {/if}
  {:else}
    <AIStatusLine
      {task}
      label="Generate briefing"
      hint="Alerts, patterns and a short briefing from your accounts, budget, money lent and recent transactions."
    />
  {/if}
</Card>
