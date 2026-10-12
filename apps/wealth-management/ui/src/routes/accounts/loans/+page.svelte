<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { api } from '$lib/api/endpoints';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Masked from '$lib/components/Masked.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatPercent } from '$lib/format';

  let { data } = $props();

  // WM-LOAN-1 summary.
  const totals = $derived(
    data.loans.reduce(
      (sum, loan) => ({
        monthlyDebt: sum.monthlyDebt + loan.monthlyDebt,
        monthlyPaid: sum.monthlyPaid + loan.monthlyPaid,
        monthlyRemaining: sum.monthlyRemaining + loan.monthlyRemaining,
        yearlyRemaining: sum.yearlyRemaining + loan.yearlyRemaining,
      }),
      { monthlyDebt: 0, monthlyPaid: 0, monthlyRemaining: 0, yearlyRemaining: 0 },
    ),
  );
  const paidPercent = $derived(totals.monthlyDebt ? (totals.monthlyPaid / totals.monthlyDebt) * 100 : 0);
  const progressColor = (percent: number) => (percent >= 100 ? 'bg-emerald-500' : 'bg-indigo-500');
</script>

<PageHeader title="Loans" subtitle="Lender names are hidden with the privacy mask" />

<div class="grid gap-4 sm:grid-cols-3">
  <Stat label="Month debt" value={totals.monthlyDebt} hint="Paid {formatPercent(paidPercent, 0)}" />
  <Stat label="Remaining this month" value={totals.monthlyRemaining} />
  <Stat label="Total yearly remaining" value={totals.yearlyRemaining} />
</div>

<!-- WM-LOAN-2 -->
<Card title="Debt tracker" class="mt-4">
  {#if data.loans.length}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[48rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Lender</th>
            <th class="px-5 py-2 text-right">Month paid / total</th>
            <th class="w-40 px-5 py-2">Progress</th>
            <th class="px-5 py-2 text-right">Yearly total</th>
            <th class="px-5 py-2 text-right">Yearly paid</th>
            <th class="px-5 py-2 text-right">Yearly remaining</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each data.loans as loan, index (index)}
            {@const percent = loan.monthlyDebt ? (loan.monthlyPaid / loan.monthlyDebt) * 100 : 0}
            <tr>
              <td class="px-5 py-2.5 font-medium">
                <Masked text={loan.name} />
                {#if loan.extra}<p class="muted text-xs font-normal">{loan.extra}</p>{/if}
              </td>
              <td class="px-5 py-2.5 text-right"
                ><Money value={loan.monthlyPaid} /> / <Money value={loan.monthlyDebt} /></td
              >
              <td class="px-5 py-2.5"
                ><ProgressBar {percent} color={progressColor(percent)} label="Paid this month" /></td
              >
              <td class="px-5 py-2.5 text-right"><Money value={loan.yearlyDebt} /></td>
              <td class="px-5 py-2.5 text-right"><Money value={loan.yearlyPaid} /></td>
              <td class="px-5 py-2.5 text-right"><Money value={loan.yearlyRemaining} /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {:else}
    <EmptyState title="No loans" detail="Loans come from the Loans sheet." />
  {/if}
</Card>

<!-- WM-LOAN-3 -->
<InsightCard
  title="AI loan review"
  class="mt-4"
  load={(force, signal) => api.review('loan-review', {}, { force, signal })}
/>
