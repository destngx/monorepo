<script lang="ts">
  import { untrack } from 'svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { SuggestedLimit } from '$lib/api/types';
  import LineChart from '$lib/charts/LineChart.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Button from '$lib/components/Button.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Money from '$lib/components/Money.svelte';
  import { DIRECTIONS, natureLabel, spendShape } from '$lib/budget/trend';
  import Badge from '$lib/components/Badge.svelte';
  import { formatMonth } from '$lib/format';
  import OpenRecommendations from '$lib/recommendations/OpenRecommendations.svelte';
  import { recordNote } from '$lib/recommendations/recommendations';
  import { base } from '$app/paths';
  import { toasts } from '$lib/state/toasts.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { severityStatus } from './severity';
  import { AITask } from './task.svelte';

  // Budget advisor (WM-BUD-5): model advice over engine-computed facts.
  const task = new AITask((force, signal) => api.budgetAdvice({ force, signal }));

  // The advisor files trackable actions to the Budget page's recommendations
  // and lists the open ones in this card (WM-BUD-11); a run that recorded
  // something re-reads the list.
  let { class: className = '' }: { class?: string } = $props();
  let recorded = $state(0);
  $effect(() => {
    // Only a new advice value triggers this; the bump itself must not.
    if (task.value?.recorded) untrack(() => recorded++);
  });

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

<Card
  class={className}
  title="AI budget advisor"
  subtitle="Limits to raise, lower or keep from your year-to-date and last 3 months, alerts and a 30-day cash projection"
>
  {#snippet actions()}
    {#if task.status === 'done'}
      <Button size="icon" icon="refresh" label="Regenerate advice" onclick={() => void task.run(true)} />
    {/if}
  {/snippet}
  {#if task.status === 'done' && task.value}
    {@const advice = task.value}
    <Markdown text={advice.briefing} />
    {#if advice.recorded}
      <p class="muted mt-1 text-xs" role="status">{recordNote(advice.recorded)}</p>
    {/if}
    {#if advice.alerts.length}
      <ul class="mt-4 grid grid-cols-1 gap-2 md:grid-cols-2">
        {#each advice.alerts as alert, index (index)}
          {@const status = severityStatus(alert.severity)}
          <li class="rounded-xl border p-3 text-sm {status.box}">
            <p class="flex items-start gap-1.5 font-semibold">
              <Icon name={status.icon} size={16} class="mt-0.5" /><span class="sr-only">{status.label}:</span>
              <span><MaskedText text={alert.title} />{alert.category ? ` · ${alert.category}` : ''}</span>
            </p>
            <p class="text-stone-700 dark:text-stone-300"><MaskedText text={alert.message} /></p>
          </li>
        {/each}
      </ul>
    {/if}
    <p class="mt-4 text-sm" data-testid="savings-check">
      {#if advice.savingsCheck.onTrack}
        <span class="font-semibold">Savings on track:</span> wants keep their usual level.
      {:else}
        <span class="font-semibold">Savings short:</span> rising or swinging wants go back to their lower average; needs are
        never cut.
      {/if}
      <span class="muted"
        >Income <Money value={advice.savingsCheck.averageIncome} /> a month, usual limits <Money
          value={advice.savingsCheck.usualSpend}
        />, saving needed <Money value={advice.savingsCheck.required} /> (goals or 30%, whichever is higher).</span
      >
    </p>
    {#if advice.suggestedLimits.length}
      <div class="mt-4 overflow-x-auto">
        <table class="w-full text-sm">
          <thead class="muted text-left text-xs">
            <tr
              ><th class="py-1 pr-3">Category</th><th class="pr-3">Move</th><th class="pr-3 text-right">Limit</th><th
                class="pr-3 text-right">YTD avg</th
              ><th class="pr-3 text-right">3-mo avg</th><th class="pr-3 text-right">Suggested</th><th class="pr-3"
                >Why</th
              ><th><span class="sr-only">Actions</span></th></tr
            >
          </thead>
          <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
            {#each advice.suggestedLimits as limit (limit.category)}
              {@const shape = spendShape(limit.trend, limit.pattern)}
              <tr>
                <td class="py-2 pr-3"
                  ><span class="font-medium">{limit.category}</span><span class="muted block text-xs"
                    >{[natureLabel(limit.nature), shape].filter(Boolean).join(' · ')}</span
                  ></td
                >
                <td class="pr-3"
                  ><Badge tone={DIRECTIONS[limit.direction].tone}>{DIRECTIONS[limit.direction].label}</Badge></td
                >
                <td class="pr-3 text-right"><Money value={limit.currentLimit} /></td>
                <td class="pr-3 text-right"><Money value={limit.yearToDateAverage} /></td>
                <td class="pr-3 text-right"><Money value={limit.averageSpend} /></td>
                <td class="pr-3 text-right font-semibold"><Money value={limit.suggestedLimit} /></td>
                <td class="muted pr-3 text-xs"><MaskedText text={limit.rationale} /></td>
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
    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-3">
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
                <li><span class="font-medium">{goal.goal}:</span> <MaskedText text={goal.impact} /></li>
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
      hint="Limits to raise, lower or keep from your year-to-date and 3-month spending, needs before wants, spikes and a cash projection."
    />
  {/if}
  <section class="mt-6 border-t border-stone-900/5 pt-4 dark:border-white/5" aria-labelledby="budget-actions">
    <div class="mb-2 flex items-center justify-between gap-2">
      <h3 id="budget-actions" class="text-sm font-semibold">Actions you're tracking</h3>
      <a class="btn-ghost px-2 text-xs" href="{base}/recommendations?page=%2Fbudget" title="All budget recommendations"
        >All</a
      >
    </div>
    <OpenRecommendations
      page="/budget"
      refresh={recorded}
      empty="Nothing to track yet. The advisor adds actions worth following over the coming weeks here."
    />
  </section>
</Card>

<Modal open={confirming !== null} title="Apply suggested limit" onclose={() => (confirming = null)} busy={applying}>
  {#if confirming}
    <p class="text-sm">
      Set the {formatMonth(adviceMonth)} limit for <strong>{confirming.category}</strong> from
      <Money value={confirming.currentLimit} /> to <Money value={confirming.suggestedLimit} />? It is written to the
      Budget sheet.
    </p>
  {/if}
  {#snippet footer()}
    <Button disabled={applying} onclick={() => (confirming = null)}>Cancel</Button>
    <Button variant="primary" busy={applying} busyText="Applying…" onclick={() => void apply()}>Apply</Button>
  {/snippet}
</Modal>
