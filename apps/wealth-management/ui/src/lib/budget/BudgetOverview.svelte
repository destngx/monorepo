<script lang="ts">
  import type { BudgetReport } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeadline from '$lib/components/PageHeadline.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatPercent } from '$lib/format';
  import { statusBar, statusLabel, statusTone } from './status';

  // WM-BUD-3 overview, headline first (ROADMAP P2): am I within budget?
  interface Props {
    report: BudgetReport;
  }

  let { report }: Props = $props();
  const burst = $derived(report.percentUsed > 100);
  const tone = $derived(burst ? 'critical' : statusTone[report.status]);
  const overPace = $derived(report.totalLimit > 0 && report.projectedSpend > report.totalLimit);
</script>

<PageHeadline question="Am I within budget?" {tone} id="budget-headline">
  {#snippet actions()}
    {#if burst}<Badge status="critical">Budget Burst</Badge>{/if}
    <Badge status={statusTone[report.status]}
      >{statusLabel[report.status]} · {formatPercent(report.percentUsed, 0)}</Badge
    >
  {/snippet}
  {#snippet answer()}
    <span><Money value={report.totalSpent} /> spent of <Money value={report.totalLimit} /></span>
  {/snippet}
  <p data-testid="budget-pace">
    {#if report.totalLimit <= 0}
      No expense limits are set for this month.
    {:else if burst}
      Over the expense budget by <Money value={report.totalSpent - report.totalLimit} />.
    {:else if report.daysRemaining === 0}
      The month ended within the expense budget.
    {:else if overPace}
      On pace to spend <Money value={report.projectedSpend} /> by the month end, over the limit.
    {:else}
      On pace to spend <Money value={report.projectedSpend} /> by the month end, within the limit.
    {/if}
  </p>
  <div class="mt-4">
    <ProgressBar percent={report.percentUsed} color={statusBar[report.status]} label="Budget used" />
  </div>
  <div class="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-4">
    <Stat label="Remaining" value={report.totalRemaining} />
    <Stat label="Daily burn" value={report.dailyBurn} />
    <Stat label="Projected spend" value={report.projectedSpend} />
    <Stat label="Days remaining">
      <span class="tabular">{report.daysRemaining}</span>
      <span class="muted text-sm font-normal">of {report.daysInMonth}</span>
    </Stat>
  </div>
</PageHeadline>
