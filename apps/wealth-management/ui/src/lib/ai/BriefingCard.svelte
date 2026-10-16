<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import { mask } from '$lib/state/mask.svelte';
  import Markdown from './Markdown.svelte';
  import { maskText } from './moneyMask';
  import { severityStatus } from './severity';
  import { AITask } from './task.svelte';

  // Dashboard intelligence briefing (WM-DASH-6, WM-AI-6).
  const task = new AITask((force, signal) => api.briefing({ force, signal }));

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
      <ul class="mb-4 grid gap-2 md:grid-cols-2">
        {#each briefing.alerts as alert, index (index)}
          {@const status = severityStatus(alert.severity)}
          <li class="rounded-xl border p-3 text-sm {status.box}">
            <p class="flex items-start gap-1.5 font-semibold">
              <Icon name={status.icon} size={16} class="mt-0.5" /><span class="sr-only">{status.label}:</span><span
                >{maskText(alert.title, mask.revealed)}</span
              >
            </p>
            <p class="text-slate-700 dark:text-slate-300">{maskText(alert.message, mask.revealed)}</p>
          </li>
        {/each}
      </ul>
    {/if}
    <Markdown text={briefing.briefing} />
    {#if briefing.patterns.length}
      <div class="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {#each briefing.patterns as pattern, index (index)}
          <div class="rounded-xl border border-slate-900/10 p-3 dark:border-white/10">
            <p class="muted text-xs font-medium tracking-wide uppercase">
              {patternLabels[pattern.kind] ?? pattern.kind}
            </p>
            <p class="mt-1 text-sm font-semibold">{maskText(pattern.title, mask.revealed)}</p>
            <Markdown text={pattern.content} class="mt-1" />
          </div>
        {/each}
      </div>
    {/if}
  {:else}
    <AIStatusLine
      {task}
      label="Generate briefing"
      hint="Alerts, patterns and a short briefing from your accounts, budget, loans and recent transactions."
    />
  {/if}
</Card>
