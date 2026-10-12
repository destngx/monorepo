<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { severityStyle } from './severity';
  import { AITask } from './task.svelte';

  // Budget advisor (WM-BUD-5): model advice over engine-computed facts.
  const task = new AITask((force, signal) => api.budgetAdvice({ force, signal }));
</script>

<Card title="AI budget advisor" subtitle="Suggested limits, alerts and a 30-day cash projection">
  {#snippet actions()}
    {#if task.status === 'done'}
      <button type="button" class="btn-ghost px-2" aria-label="Regenerate advice" onclick={() => task.run(true)}
        ><Icon name="refresh" size={16} /></button
      >
    {/if}
  {/snippet}
  {#if task.status === 'done' && task.value}
    {@const advice = task.value}
    <Markdown text={advice.briefing} />
    {#if advice.alerts.length}
      <ul class="mt-4 grid gap-2 md:grid-cols-2">
        {#each advice.alerts as alert, index (index)}
          <li class="rounded-xl border p-3 text-sm {severityStyle[alert.severity]}">
            <p class="font-semibold">{alert.title}{alert.category ? ` · ${alert.category}` : ''}</p>
            <p class="text-slate-700 dark:text-slate-300">{alert.message}</p>
          </li>
        {/each}
      </ul>
    {/if}
    {#if advice.suggestedLimits.length}
      <div class="mt-4 overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="muted text-left text-xs">
            <tr
              ><th class="py-1 pr-3">Category</th><th class="pr-3 text-right">Limit</th><th class="pr-3 text-right"
                >3-mo avg</th
              ><th class="pr-3 text-right">Suggested</th><th>Why</th></tr
            >
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each advice.suggestedLimits as limit (limit.category)}
              <tr>
                <td class="py-2 pr-3 font-medium">{limit.category}</td>
                <td class="pr-3 text-right"><Money value={limit.currentLimit} /></td>
                <td class="pr-3 text-right"><Money value={limit.averageSpend} /></td>
                <td class="pr-3 text-right font-semibold"><Money value={limit.suggestedLimit} /></td>
                <td class="muted text-xs">{limit.rationale}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
    <div class="mt-4 grid gap-4 lg:grid-cols-3">
      <div class="lg:col-span-2">
        <p class="muted mb-1 text-xs">Projected cash, next 30 days</p>
        <LineChart
          points={advice.projection.map((point) => ({ date: point.date, value: point.balance }))}
          label="30-day cash projection"
        />
      </div>
      <div class="space-y-3 text-sm">
        <div>
          <p class="muted text-xs">Forecasted savings this month</p>
          <p class="text-lg font-semibold"><Money value={advice.forecastedSavings} /></p>
        </div>
        {#if advice.unusualVelocity.length}
          <div>
            <p class="muted text-xs">Spending faster than usual</p>
            <ul>
              {#each advice.unusualVelocity as flag (flag.category)}
                <li>{flag.category}: {flag.ratio.toFixed(1)}× usual pace</li>
              {/each}
            </ul>
          </div>
        {/if}
        {#if advice.goalImpact.length}
          <div>
            <p class="muted text-xs">Goal impact</p>
            <ul class="space-y-1">
              {#each advice.goalImpact as goal (goal.goal)}
                <li><span class="font-medium">{goal.goal}:</span> {goal.impact}</li>
              {/each}
            </ul>
          </div>
        {/if}
      </div>
    </div>
  {:else}
    <AIStatusLine
      {task}
      label="Get advice"
      hint="Suggested limits from your 3-month averages, spending spikes and a cash projection."
    />
  {/if}
</Card>
