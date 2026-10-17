<script lang="ts">
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import type { FinancialHealth, HealthComponent } from '$lib/api/types';
  import BarChart from '$lib/charts/BarChart.svelte';
  import Radar from '$lib/charts/Radar.svelte';
  import { NOMINAL, REAL } from '$lib/charts/palette';
  import Ring from '$lib/charts/Ring.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatPercent } from '$lib/format';
  import AllocationSummary from '$lib/investments/AllocationSummary.svelte';
  import NetWorthHistoryCard from '$lib/networth/NetWorthHistoryCard.svelte';
  import { pillarText, projectionRows } from '$lib/health';
  import { STATUS, type StatusTone } from '$lib/status';

  // The header renders at once; the score (with its pillars, math, projection
  // and orders) and the net worth history stream in separately, so the
  // history's snapshot read never holds the score.
  let { data } = $props();
  const health = track(() => data.health);

  // WM-HEA-1 bands: red < 50, yellow 50-75, green > 75.
  const band: Record<FinancialHealth['band'], { tone: StatusTone; label: string }> = {
    red: { tone: 'critical', label: 'Needs attention' },
    yellow: { tone: 'warning', label: 'Fair' },
    green: { tone: 'good', label: 'Healthy' },
  };

  const axesOf = (value: FinancialHealth) =>
    value.components.map((c) => ({ label: c.label, fraction: c.maxScore ? c.score / c.maxScore : 0 }));

  let openMath = $state<string | null>(null);
  const formatValue = (c: HealthComponent) =>
    c.unit === '%'
      ? formatPercent(c.value)
      : c.unit === 'months'
        ? `${c.value.toFixed(1)} months`
        : `${c.value.toFixed(2)} ${c.unit}`;
  const formatTarget = (c: HealthComponent) =>
    c.unit === '%' ? formatPercent(c.target, 0) : c.unit === 'months' ? `${c.target} months` : `${c.target} ${c.unit}`;
</script>

<PageHeader
  title="Financial Health"
  subtitle={health.value ? `Computed from the last ${health.value.lookbackMonths} months of transactions` : undefined}
/>

<Deferred data={data.health} label="the health score">
  {#snippet children(value)}
    {@const status = band[value.band]}
    <div>
      {#if value.insufficientIncome}
        <div class="glass mb-4 flex items-start gap-2 border p-3 text-sm {STATUS.warning.box}" role="status">
          <Icon name={STATUS.warning.icon} size={16} class="mt-0.5" />
          <span>Not enough income history to score savings and debt reliably; those pillars may read low.</span>
        </div>
      {/if}

      <div class="grid gap-4 lg:grid-cols-3">
        <Card title="Health score">
          <div class="flex flex-col items-center gap-3">
            <Ring
              percent={value.score}
              color={STATUS[status.tone].chart}
              size={170}
              thickness={14}
              label="Health score"
            >
              <span class="text-4xl font-semibold">{Math.round(value.score)}</span>
              <span class="muted text-xs">of 100</span>
            </Ring>
            <Badge status={status.tone}>{status.label}</Badge>
          </div>
        </Card>
        <Card title="Pillars" class="lg:col-span-2">
          <div class="flex justify-center"><Radar axes={axesOf(value)} label="Health pillars" /></div>
        </Card>
      </div>

      <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat label="Liquid assets" value={value.liquidAssets} />
        <Stat label="Avg monthly income" value={value.avgMonthlyIncome} />
        <Stat label="Avg monthly expense" value={value.avgMonthlyExpense} />
        <Stat label="Freedom runway">
          <span class="tabular">{value.freedomMonths === null ? '-' : `${value.freedomMonths.toFixed(1)} months`}</span>
        </Stat>
      </div>

      <!-- WM-HEA-3: show the math -->
      <Card title="How the score is built" class="mt-4">
        <ul class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each value.components as component (component.key)}
            {@const text = pillarText(component, value)}
            {@const open = openMath === component.key}
            <li class="py-3">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p class="font-medium">{text.label}</p>
                  <p class="muted text-xs">{formatValue(component)} · target {formatTarget(component)}</p>
                  {#if text.explanation}<p class="muted mt-0.5 text-xs"><MaskedText text={text.explanation} /></p>{/if}
                </div>
                <div class="flex items-center gap-3">
                  <span class="tabular text-sm font-semibold">{component.score.toFixed(1)} / {component.maxScore}</span>
                  <Button
                    size="sm"
                    aria-expanded={open}
                    title="{open ? 'Hide' : 'Show'} how {text.label} is scored"
                    onclick={() => (openMath = open ? null : component.key)}
                    >{open ? 'Hide the math' : 'Show the math'}</Button
                  >
                </div>
              </div>
              {#if open}
                <div class="mt-2 rounded-xl bg-slate-900/5 p-3 text-sm dark:bg-white/5">
                  <p class="font-mono text-xs"><MaskedText text={component.formula} /></p>
                  <p class="muted mt-1"><MaskedText text={component.detail} /></p>
                </div>
              {/if}
            </li>
          {/each}
        </ul>
      </Card>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <div class="grid gap-4 lg:grid-cols-3">
        <SkeletonCard title="Health score" chart={170} lines={0} />
        <SkeletonCard title="Pillars" chart={170} lines={0} class="lg:col-span-2" />
      </div>
      <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
        {#each ['liquid assets', 'average monthly income', 'average monthly expense', 'freedom runway'] as label (label)}
          <SkeletonStat {label} />
        {/each}
      </div>
      <SkeletonCard title="How the score is built" rows={5} lines={0} class="mt-4" />
    </div>
  {/snippet}
</Deferred>

<!-- ROADMAP P3: net worth history from the monthly snapshots, streamed apart from the score. -->
<div id="net-worth-history" class="mt-4 scroll-mt-4">
  <Deferred data={data.netWorthHistory} label="the net worth history">
    {#snippet children(history)}<NetWorthHistoryCard {history} />{/snippet}
    {#snippet skeleton()}<SkeletonCard title="Net worth history" chart={240} lines={1} />{/snippet}
  </Deferred>
</div>

<!-- WM-HEA-2, WM-RULE-12, the doctor's orders and the allocation come with the score. -->
<Deferred data={data.health} label="the projection and the doctor's orders" class="mt-4">
  {#snippet children(value)}
    {@const projection = projectionRows(value.projection)}
    <div>
      {#if projection.length}
        <Card title="Net worth projection" subtitle="Real values are in today's money, after inflation" class="mb-4">
          <div class="grid items-center gap-4 lg:grid-cols-2">
            <BarChart
              labels={projection.map((row) => `${row.years}y`)}
              series={[
                { name: 'Nominal', color: NOMINAL, values: projection.map((row) => row.nominal) },
                { name: 'Real', color: REAL, values: projection.map((row) => row.real) },
              ]}
              height={200}
              label="Projected net worth"
            />
            <table class="w-full text-sm">
              <thead>
                <tr class="muted text-left text-xs uppercase">
                  <th class="py-2">Horizon</th>
                  <th class="py-2 text-right">Nominal</th>
                  <th class="py-2 text-right">Real</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
                {#each projection as row (row.years)}
                  <tr>
                    <td class="py-2">{row.years} years</td>
                    <td class="py-2 text-right"><Money value={row.nominal} /></td>
                    <td class="py-2 text-right"><Money value={row.real} /></td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </Card>
      {/if}

      <div class="grid gap-4 lg:grid-cols-2">
        <Card title="Doctor's orders">
          {#if value.recommendations.length}
            <ol class="space-y-3">
              {#each value.recommendations as rec, i (i)}
                <li class="flex gap-3">
                  <span class="grid size-6 shrink-0 place-items-center rounded-full bg-indigo-600 text-xs text-white"
                    >{i + 1}</span
                  >
                  <div>
                    <p class="font-medium"><MaskedText text={rec.title} /></p>
                    <p class="muted text-sm"><MaskedText text={rec.action} /></p>
                  </div>
                </li>
              {/each}
            </ol>
          {:else}
            <p class="muted text-sm">No recommendations: every pillar is on target.</p>
          {/if}
        </Card>
        <!-- ROADMAP C4: the same allocation as Investments; diversification scores it. -->
        <AllocationSummary allocation={value.allocation} />
      </div>
    </div>
  {/snippet}
  <!-- The score's section above already shows this request's error and Retry. -->
  {#snippet error()}{/snippet}
  {#snippet skeleton()}
    <div class="grid gap-4 lg:grid-cols-2">
      <SkeletonCard title="Doctor's orders" lines={4} />
      <SkeletonCard title="Asset allocation" chart={140} lines={0} />
    </div>
  {/snippet}
</Deferred>

<!-- WM-AI-14 -->
<RecommendationsPanel
  title="Recommendations"
  subtitle="From the health review: what the score means and what to do next"
  page="/health"
  class="mt-4"
  review={(force, signal) => api.healthCommentary({ force, signal })}
/>
