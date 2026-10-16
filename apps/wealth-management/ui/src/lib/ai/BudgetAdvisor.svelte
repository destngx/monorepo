<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { SuggestedLimit } from '$lib/api/types';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatMonth } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { maskText } from './moneyMask';
  import { severityStatus } from './severity';
  import { AITask } from './task.svelte';

  // Budget advisor (WM-BUD-5): model advice over engine-computed facts.
  const task = new AITask((force, signal) => api.budgetAdvice({ force, signal }));

  // "Apply" writes a suggested limit to the Budget sheet for the advice's month (WM-BUD-6).
  let confirming = $state<SuggestedLimit | null>(null);
  let applying = $state(false);
  let applied = $state<string[]>([]);
  const adviceMonth = $derived(task.value?.month ?? '');

  async function apply() {
    const limit = confirming;
    if (!limit || !adviceMonth) return;
    applying = true;
    try {
      await api.updateBudgetLimit({
        year: Number(adviceMonth.slice(0, 4)),
        category: limit.category,
        month: adviceMonth,
        limit: limit.suggestedLimit,
      });
      applied = [...applied, limit.category];
      toasts.push('success', `Limit for ${limit.category} updated.`);
      confirming = null;
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not apply the suggested limit');
    } finally {
      applying = false;
    }
  }
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
          {@const status = severityStatus(alert.severity)}
          <li class="rounded-xl border p-3 text-sm {status.box}">
            <p class="flex items-start gap-1.5 font-semibold">
              <Icon name={status.icon} size={16} class="mt-0.5" /><span class="sr-only">{status.label}:</span>
              <span>{maskText(alert.title, mask.revealed)}{alert.category ? ` · ${alert.category}` : ''}</span>
            </p>
            <p class="text-slate-700 dark:text-slate-300">{maskText(alert.message, mask.revealed)}</p>
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
              ><th class="pr-3 text-right">Suggested</th><th class="pr-3">Why</th><th
                ><span class="sr-only">Actions</span></th
              ></tr
            >
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each advice.suggestedLimits as limit (limit.category)}
              <tr>
                <td class="py-2 pr-3 font-medium">{limit.category}</td>
                <td class="pr-3 text-right"><Money value={limit.currentLimit} /></td>
                <td class="pr-3 text-right"><Money value={limit.averageSpend} /></td>
                <td class="pr-3 text-right font-semibold"><Money value={limit.suggestedLimit} /></td>
                <td class="muted pr-3 text-xs">{maskText(limit.rationale, mask.revealed)}</td>
                <td class="text-right">
                  {#if applied.includes(limit.category)}
                    <span class="inline-flex items-center gap-1 text-xs text-emerald-700 dark:text-emerald-300"
                      ><Icon name="check" size={14} /> Applied</span
                    >
                  {:else}
                    <button
                      type="button"
                      class="btn-ghost px-2 py-1 text-xs"
                      disabled={limit.suggestedLimit === limit.currentLimit}
                      onclick={() => (confirming = limit)}>Apply</button
                    >
                  {/if}
                </td>
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
                <li><span class="font-medium">{goal.goal}:</span> {maskText(goal.impact, mask.revealed)}</li>
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

<Modal open={confirming !== null} title="Apply suggested limit" onclose={() => (confirming = null)}>
  {#if confirming}
    <p class="text-sm">
      Set the {formatMonth(adviceMonth)} limit for <strong>{confirming.category}</strong> from
      <Money value={confirming.currentLimit} /> to <Money value={confirming.suggestedLimit} />? It is written to the
      Budget sheet.
    </p>
  {/if}
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (confirming = null)}>Cancel</button>
    <button type="button" class="btn-primary" disabled={applying} onclick={apply}
      >{applying ? 'Applying…' : 'Apply'}</button
    >
  {/snippet}
</Modal>
