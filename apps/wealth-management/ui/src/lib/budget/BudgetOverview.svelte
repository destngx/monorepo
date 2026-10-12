<script lang="ts">
  import type { BudgetReport } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatPercent } from '$lib/format';
  import { statusBar, statusLabel, statusTone } from './status';

  // WM-BUD-3 overview.
  interface Props {
    report: BudgetReport;
  }

  let { report }: Props = $props();
</script>

<Card>
  <div class="flex flex-wrap items-end justify-between gap-4">
    <div>
      <p class="muted text-xs tracking-wide uppercase">Spent of expense budget</p>
      <p class="mt-1 text-2xl font-semibold">
        <Money value={report.totalSpent} />
        <span class="muted text-base font-normal">/ <Money value={report.totalLimit} /></span>
      </p>
    </div>
    <div class="flex items-center gap-2">
      {#if report.percentUsed > 100}<Badge tone="red">Budget Burst</Badge>{/if}
      <Badge tone={statusTone[report.status]}
        >{statusLabel[report.status]} · {formatPercent(report.percentUsed, 0)}</Badge
      >
    </div>
  </div>
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
</Card>
