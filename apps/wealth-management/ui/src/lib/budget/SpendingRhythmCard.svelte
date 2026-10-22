<script lang="ts">
  import type { SpendingWindows } from '$lib/api/types';
  import BarChart from '$lib/charts/BarChart.svelte';
  import { DAY_TO_DAY, FIXED } from '$lib/charts/palette';
  import Badge from '$lib/components/Badge.svelte';
  import { Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent, formatSignedPercent } from '$lib/format';
  import {
    recommendationKey,
    recommendationStatus,
    SPENDING_RANGES,
    spendingDays,
    spendingWindow,
    type SpendingRange,
  } from './spending';
  import { natureLabel } from './trend';

  // How am I spending? (WM-BUD-12): today, the last 30 days and the last 3 months, each ending
  // today, with the engine's advice. Day-to-day averages leave fixed costs out (paid once a month).
  interface Props {
    spending: SpendingWindows;
  }

  let { spending }: Props = $props();

  let range = $state<SpendingRange>('30d');
  const recent = $derived(spending.windows[0]);
  const months = $derived(spending.windows[1]);
  const shown = $derived(spendingWindow(spending, range));
  const chart = $derived(spendingDays(spending.daily, range));
  const hasSpend = $derived(months.spent > 0);
</script>

<Card
  title="How am I spending?"
  subtitle="Today, the last 30 days and the last 3 months, to {formatDate(spending.asOf)}"
>
  <div class="grid grid-cols-1 gap-4 sm:grid-cols-3" data-testid="spending-stats">
    <Stat label="Today" value={spending.today.spent} change={spending.today.changePercent} invert>
      {#snippet hint()}vs a typical day of <Money value={spending.today.typicalDay} />{/snippet}
    </Stat>
    <Stat label="Last 30 days" value={recent.spent} change={recent.dailyAverageChangePercent} invert>
      {#snippet hint()}<Money value={recent.dailyAverage} /> a day, day to day, vs the 30 days before{/snippet}
    </Stat>
    <Stat label="Last 3 months" value={months.spent} change={months.dailyAverageChangePercent} invert>
      {#snippet hint()}<Money value={months.dailyAverage} /> a day, about <Money value={months.monthlyAverage} /> a month{/snippet}
    </Stat>
  </div>

  <h3 class="mt-6 mb-2 text-sm font-semibold">What to do</h3>
  {#if spending.recommendations.length}
    <ul class="space-y-2 text-sm" data-testid="spending-advice">
      {#each spending.recommendations as advice (recommendationKey(advice))}
        {@const status = recommendationStatus(advice)}
        {@const percent = formatPercent(Math.abs(advice.changePercent), 0)}
        <li class="flex items-start gap-2">
          <Badge status={status.tone}>{status.label}</Badge>
          <p>
            {#if advice.kind === 'faster'}
              Day-to-day spending is up {percent} on your 3-month average: <Money value={advice.current} /> a day against
              <Money value={advice.usual} />, about <Money value={advice.amount} /> more a month. Rein in the rising categories.
            {:else if advice.kind === 'slower'}
              Day-to-day spending is down {percent} on your 3-month average: <Money value={advice.current} /> a day against
              <Money value={advice.usual} />. About <Money value={advice.amount} /> a month is free to move to savings or
              goals.
            {:else if advice.kind === 'split_rising'}
              <span class="font-medium">#{advice.tag}</span> in {advice.category} is up {percent}: <Money
                value={advice.current}
              /> in the last 30 days against a usual <Money value={advice.usual} />. A habit worth trimming: set
              yourself a weekly amount for it (about <Money value={advice.amount} /> a month over now).
            {:else if advice.kind === 'category_rising'}
              <span class="font-medium">{advice.category}</span> is up {percent}: <Money value={advice.current} /> in the
              last 30 days against a usual <Money value={advice.usual} />.
              {#if advice.nature === 'essential'}
                It is a need, so do not cut it blindly: check what changed (prices, a one-off buy).
              {:else if !advice.nature}
                Cut back towards your usual level, and tag its nature in Categories so advice can tell a need from a
                want.
              {:else}
                Cut back towards your usual level, about <Money value={advice.amount} /> a month.
              {/if}
            {:else if advice.kind === 'heavy_day'}
              Today's day-to-day spending is <Money value={advice.current} />, {formatPercent(advice.changePercent, 0)} over
              a typical day (<Money value={advice.usual} />). Go lighter over the next few days to even it out.
            {/if}
          </p>
        </li>
      {/each}
    </ul>
  {:else if hasSpend}
    <p class="muted text-sm" data-testid="spending-advice">
      Spending is in line with your usual: nothing stands out in the last 30 days.
    </p>
  {:else}
    <p class="muted text-sm" data-testid="spending-advice">No expense in the last 3 months yet.</p>
  {/if}

  <div class="mt-6 mb-2 flex flex-wrap items-center justify-between gap-2">
    <h3 class="text-sm font-semibold">Daily spend and categories</h3>
    <div class="flex gap-1" role="group" aria-label="Window">
      {#each SPENDING_RANGES as option (option.value)}
        <button
          type="button"
          class="btn-ghost rounded-md px-2 py-0.5 text-xs {range === option.value
            ? 'bg-accent-500/15 font-semibold text-accent-700 hover:bg-accent-500/20 dark:text-accent-300'
            : 'muted'}"
          aria-pressed={range === option.value}
          onclick={() => (range = option.value)}>{option.label}</button
        >
      {/each}
    </div>
  </div>
  <BarChart
    labels={chart.labels}
    series={[
      { name: 'Day to day', color: DAY_TO_DAY, values: chart.days.map((day) => day.dayToDay) },
      { name: 'Fixed costs', color: FIXED, values: chart.days.map((day) => day.fixed) },
    ]}
    stacked
    height={180}
    label="Daily spend, {range === '30d' ? 'last 30 days' : 'last 3 months'}"
  />
  <p class="muted mt-1 text-xs" data-testid="spending-window-summary">
    <Money value={shown.spent} /> spent: <Money value={shown.dayToDay} /> day to day and <Money value={shown.fixed} /> fixed
    costs. Day-to-day average <Money value={shown.dailyAverage} /> a day{#if shown.dailyAverageChangePercent !== null},
      {formatSignedPercent(shown.dailyAverageChangePercent)} on the {shown.days} days before{/if}.
  </p>

  {#if shown.categories.length}
    <div class="mt-4 overflow-x-auto">
      <table class="w-full text-sm" data-testid="spending-categories">
        <thead class="muted text-left text-xs">
          <tr>
            <th class="py-1.5 font-medium">Category</th>
            <th class="py-1.5 text-right font-medium">Spent</th>
            <th class="py-1.5 text-right font-medium">A month</th>
            <th class="py-1.5 text-right font-medium">Share</th>
            <th class="py-1.5 text-right font-medium">vs {shown.days} days before</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-stone-900/5 dark:divide-white/5">
          {#each shown.categories as category (category.category)}
            <tr>
              <td class="py-1.5">
                {category.category} <span class="muted text-xs">{natureLabel(category.nature)}</span>
                {#each category.splits ?? [] as split (split.tag)}
                  <span class="muted block text-xs"
                    >of which <span class="font-medium">#{split.tag}</span>
                    <Money value={split.spent} /> ({formatPercent(split.percentOfSpent, 0)})</span
                  >
                {/each}
              </td>
              <td class="tabular py-1.5 text-right"><Money value={category.spent} /></td>
              <td class="tabular py-1.5 text-right"><Money value={category.monthlyAverage} /></td>
              <td class="tabular py-1.5 text-right">{formatPercent(category.percentOfSpent, 0)}</td>
              <td class="tabular py-1.5 text-right"
                >{category.changePercent === null ? 'new' : formatSignedPercent(category.changePercent, 0)}</td
              >
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</Card>
