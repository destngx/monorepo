<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import ChartInsight from '$lib/ai/ChartInsight.svelte';
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import type { YearlyCashFlow } from '$lib/api/types';
  import CategoryMonthsTable from '$lib/cashflow/CategoryMonthsTable.svelte';
  import ExpenseSplitCard from '$lib/cashflow/ExpenseSplitCard.svelte';
  import SavingsRateChart from '$lib/cashflow/SavingsRateChart.svelte';
  import { comparisonSpan, projectionBasis, verdictMessage } from '$lib/cashflow/yearly';
  import BarChart from '$lib/charts/BarChart.svelte';
  import { EXPENSE, INCOME } from '$lib/charts/palette';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonHeadline from '$lib/components/skeleton/SkeletonHeadline.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import { currentMonth, formatMonth, formatPercent } from '$lib/format';

  // The header and year switcher render at once (the year is in the URL);
  // the report streams in and the skeleton returns only for another year.
  let { data } = $props();
  const tracked = track(() => data.report, { scope: () => data.year });
  const thisYear = Number(currentMonth().slice(0, 4));
  // Years with data once the report answers; until then the year asked for.
  const years = $derived(tracked.value?.years ?? [data.year]);

  // What the report says, worked out once per report.
  function view(report: YearlyCashFlow) {
    // WM-DASH-8: compared with the same months of the previous year.
    const span = comparisonSpan(report.year, report.monthsElapsed);
    return {
      hint: span ? ` vs ${span}` : undefined,
      averageOver:
        report.monthsElapsed === 12
          ? 'per month'
          : report.monthsElapsed > 0
            ? `per month over ${report.monthsElapsed} month${report.monthsElapsed === 1 ? '' : 's'}`
            : 'no months yet',
      verdict: verdictMessage(report.year, report.verdict, report.savingsTarget, report.monthsElapsed),
      // Room to save leaves out the month under way, like the engine.
      completedMonths: report.year === thisYear ? report.monthsElapsed - 1 : report.monthsElapsed,
    };
  }

  let breakdown = $state<'expense' | 'income'>('expense');

  function goYear(year: number) {
    const url = new URL(page.url);
    if (year === thisYear) url.searchParams.delete('year');
    else url.searchParams.set('year', String(year));
    void goto(url, { keepFocus: true, noScroll: true });
  }
</script>

<PageHeader title="Cash Flow" subtitle="Income and expense by month, {data.year}">
  {#snippet actions()}
    <div class="flex items-center gap-1">
      <Button size="icon" icon="chevronLeft" label="Previous year" onclick={() => goYear(data.year - 1)} />
      <select
        class="field w-auto py-1.5"
        aria-label="Year"
        value={data.year}
        onchange={(event) => goYear(Number(event.currentTarget.value))}
      >
        {#each years.includes(data.year) ? years : [data.year, ...years] as year (year)}
          <option value={year}>{year}</option>
        {/each}
      </select>
      <Button size="icon" icon="chevronRight" label="Next year" onclick={() => goYear(data.year + 1)} />
      <Button disabled={data.year === thisYear} onclick={() => goYear(thisYear)}>This year</Button>
    </div>
  {/snippet}
</PageHeader>

<Deferred data={data.report} scope={data.year} label="the cash flow for {data.year}">
  {#snippet children(report)}
    {@const v = view(report)}
    <div>
      <div class="mb-4 grid gap-4 {report.projection ? 'lg:grid-cols-2' : ''}">
        <PageHeadline
          question="Am I okay? {report.year}"
          tone={v.verdict.tone}
          title={v.verdict.title}
          id="verdict-title"
        >
          <p>{v.verdict.text}</p>
          {#if v.verdict.detail}<p class="muted mt-1">{v.verdict.detail}</p>{/if}
        </PageHeadline>
        {#if report.projection}
          {@const projection = report.projection}
          <section class="glass p-5" aria-labelledby="projection-title">
            <p class="muted text-xs font-medium uppercase">If the year keeps this pace</p>
            <h2 id="projection-title" class="mt-1 text-lg font-semibold">
              <Money value={projection.cashFlow} signed tone /> by December
            </h2>
            <p class="mt-1 text-sm">
              <Money value={projection.income} /> in, <Money value={projection.expense} /> out,
              {projection.income > 0 ? `keeping ${formatPercent(projection.savingsRate)}.` : 'no income on record.'}
            </p>
            <p class="muted mt-1 text-sm">{projectionBasis(projection.basisMonths)}</p>
          </section>
        {/if}
      </div>

      {#if report.monthsElapsed > 0}
        <RecommendationsPanel
          title="Recommendations"
          subtitle="From the {report.year} year review"
          page="/cash-flow"
          class="mb-4"
          hint="The year review says what went well and what to watch, and adds what to do next to your recommendations."
          review={(force, signal) => api.review('year-review', { year: report.year }, { force, signal })}
        />
      {/if}

      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        <Stat label="Income" value={report.total.income} change={report.incomeChangePercent} hint={v.hint} />
        <Stat label="Expense" value={report.total.expense} change={report.expenseChangePercent} invert hint={v.hint} />
        <Stat label="Cash flow">
          <Money value={report.total.cashFlow} signed tone />
        </Stat>
        <Stat label="Savings rate">
          <span class="tabular">{formatPercent(report.total.savingsRate)}</span>
        </Stat>
        <Stat label="Avg income" value={report.monthlyAverage.income} hint={v.averageOver} />
        <Stat label="Avg expense" value={report.monthlyAverage.expense} hint={v.averageOver} />
      </div>

      <Card title="Income vs expense" subtitle="Monthly, {report.year}" class="mt-4">
        {#snippet actions()}
          <ChartInsight
            chartType="bar"
            title="Income vs expense {report.year}"
            timeframe={String(report.year)}
            data={report.months.map((m) => ({ month: m.month, income: m.income, expense: m.expense }))}
          />
        {/snippet}
        <BarChart
          labels={report.months.map((m) => formatMonth(m.month, true))}
          series={[
            { name: 'Income', color: INCOME, values: report.months.map((m) => m.income) },
            { name: 'Expense', color: EXPENSE, values: report.months.map((m) => m.expense) },
          ]}
          label="Income and expense per month of {report.year}"
        />
      </Card>

      <div class="mt-4 grid gap-4 {report.spikes.length ? 'lg:grid-cols-3' : ''}">
        <Card
          title="Savings rate"
          subtitle="Share of each month's income you kept"
          class={report.spikes.length ? 'lg:col-span-2' : ''}
        >
          <SavingsRateChart months={report.months} monthsElapsed={report.monthsElapsed} target={report.savingsTarget} />
        </Card>
        {#if report.spikes.length}
          <Card title="Unusual months" subtitle="Spending well above a typical month">
            <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
              {#each report.spikes as spike (spike.month)}
                <li class="py-2.5 first:pt-0 last:pb-0">
                  <p class="flex items-baseline justify-between gap-2">
                    <span class="font-medium">{formatMonth(spike.month)}</span>
                    <span class="tabular text-orange-700 dark:text-orange-300">{spike.ratio.toFixed(1)}x usual</span>
                  </p>
                  <p class="muted mt-0.5">
                    Spent <Money value={spike.expense} /> vs a typical <Money value={spike.typical} />.
                    {#if spike.category}
                      Mostly {spike.category}: <Money value={spike.categoryAmount} />, usually <Money
                        value={spike.categoryTypical}
                      />.
                    {/if}
                  </p>
                </li>
              {/each}
            </ul>
          </Card>
        {/if}
      </div>

      <ExpenseSplitCard split={report.expenseSplit} months={report.months.slice(0, v.completedMonths)} />

      <Card title="By month" class="mt-4">
        <div class="-mx-5 overflow-x-auto">
          <table class="w-full min-w-[36rem] text-sm">
            <thead>
              <tr class="muted border-b border-slate-900/10 text-left text-xs uppercase dark:border-white/10">
                <th class="px-5 py-2 font-medium">Month</th>
                <th class="px-5 py-2 text-right font-medium">Income</th>
                <th class="px-5 py-2 text-right font-medium">Expense</th>
                <th class="px-5 py-2 text-right font-medium">Cash flow</th>
                <th class="px-5 py-2 text-right font-medium">Savings rate</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
              {#each report.months as month, i (month.month)}
                {#if i < report.monthsElapsed}
                  <tr>
                    <td class="px-5 py-2">{formatMonth(month.month)}</td>
                    <td class="px-5 py-2 text-right"><Money value={month.income} /></td>
                    <td class="px-5 py-2 text-right"><Money value={month.expense} /></td>
                    <td class="px-5 py-2 text-right"><Money value={month.cashFlow} signed tone /></td>
                    <td class="tabular px-5 py-2 text-right"
                      >{month.income > 0 ? formatPercent(month.savingsRate) : '-'}</td
                    >
                  </tr>
                {:else}
                  <tr class="muted">
                    <td class="px-5 py-2">{formatMonth(month.month)}</td>
                    <td class="px-5 py-2 text-right" colspan="4">Not yet</td>
                  </tr>
                {/if}
              {/each}
            </tbody>
            <tfoot class="border-t border-slate-900/10 font-medium dark:border-white/10">
              <tr>
                <td class="px-5 py-2">Total</td>
                <td class="px-5 py-2 text-right"><Money value={report.total.income} /></td>
                <td class="px-5 py-2 text-right"><Money value={report.total.expense} /></td>
                <td class="px-5 py-2 text-right"><Money value={report.total.cashFlow} signed tone /></td>
                <td class="tabular px-5 py-2 text-right">{formatPercent(report.total.savingsRate)}</td>
              </tr>
              <tr class="muted font-normal">
                <td class="px-5 py-2">Monthly average</td>
                <td class="px-5 py-2 text-right"><Money value={report.monthlyAverage.income} /></td>
                <td class="px-5 py-2 text-right"><Money value={report.monthlyAverage.expense} /></td>
                <td class="px-5 py-2 text-right"><Money value={report.monthlyAverage.cashFlow} signed /></td>
                <td class="px-5 py-2"></td>
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>

      <Card title="By category" subtitle="Each category per month, largest first" class="mt-4">
        {#snippet actions()}
          <div class="flex rounded-xl bg-slate-900/5 p-0.5 text-xs dark:bg-white/5" role="group" aria-label="Breakdown">
            {#each [['expense', 'Expense'], ['income', 'Income']] as [value, text] (value)}
              <button
                type="button"
                aria-pressed={breakdown === value}
                class="rounded-lg px-2.5 py-1 {breakdown === value
                  ? 'bg-white shadow-sm dark:bg-slate-700'
                  : 'muted hover:bg-white/60 hover:text-slate-900 dark:hover:bg-white/10 dark:hover:text-white'}"
                onclick={() => (breakdown = value as 'expense' | 'income')}>{text}</button
              >
            {/each}
          </div>
        {/snippet}
        <CategoryMonthsTable
          year={report.year}
          rows={breakdown === 'expense' ? report.expense : report.income}
          monthsElapsed={report.monthsElapsed}
        />
      </Card>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonHeadline question="Am I okay? {data.year}" label="the cash flow" class="mb-4" />
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-6">
        {#each { length: 6 }, index (index)}<SkeletonStat label="the cash flow" />{/each}
      </div>
      <SkeletonCard title="Income vs expense" lines={0} chart={260} label="the cash flow" class="mt-4" />
      <SkeletonCard title="By month" rows={6} lines={0} label="the cash flow" class="mt-4" />
    </div>
  {/snippet}
</Deferred>
