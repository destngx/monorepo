<script lang="ts">
  import type { FundPerformance } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate, formatPercent, formatSignedPercent } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { COST_STATUS_LABEL, depositRateLabel, formatPoints, fundHeadline, VS_DEPOSIT } from './fundPerformance';
  import { quantity } from './holdings';

  // ROADMAP M3 (WM-INV-21/22): the headline answer first, then each fund holding.
  let { performance }: { performance: FundPerformance } = $props();

  const headline = $derived(fundHeadline(performance));
  const rateLabel = $derived(depositRateLabel(performance));
  const totals = $derived(performance.totals);
</script>

<Card title="Funds vs a bank deposit" subtitle="Fund certificates since purchase, from the History column">
  {#snippet actions()}
    {#if headline.kind === 'compare'}
      <Badge status={VS_DEPOSIT[headline.vs].tone}>{VS_DEPOSIT[headline.vs].label}</Badge>
    {/if}
  {/snippet}

  {#if headline.kind === 'empty'}
    <EmptyState title="No fund certificate holdings" />
  {:else}
    <div
      class="border-l-4 pl-3 {headline.kind === 'compare'
        ? STATUS[VS_DEPOSIT[headline.vs].tone].edge
        : STATUS.neutral.edge}"
    >
      {#if headline.kind === 'compare'}
        <p class="text-lg" data-testid="fund-headline">
          {#if headline.total.annualisedReturn !== null}
            Your funds earned <strong class="tabular">{formatPercent(headline.total.annualisedReturn, 2)}</strong> a year;
          {:else}
            Your funds gained <strong><Money value={headline.total.gain} signed /></strong>
            ({formatSignedPercent(headline.total.simpleReturn, 2)}, too recent to annualise);
          {/if}
          a deposit at <strong class="tabular">{formatPercent(headline.rate, 2)}</strong> would have earned
          <strong><Money value={headline.total.depositGain} /></strong>. Difference:
          <strong><Money value={headline.total.differenceVND} signed /></strong
          >{#if headline.total.differencePoints !== null}
            ({formatPoints(headline.total.differencePoints)}){/if}.
        </p>
      {:else if headline.kind === 'no_deposit'}
        <p class="text-lg" data-testid="fund-headline">
          Your funds gained <strong><Money value={headline.total.gain} signed /></strong>
          {#if headline.total.annualisedReturn !== null}
            (<strong class="tabular">{formatPercent(headline.total.annualisedReturn, 2)}</strong> a year){/if}.
          <span class="muted text-sm">No deposit comparison: {headline.issue}.</span>
        </p>
      {:else}
        <p class="text-lg" data-testid="fund-headline">
          Cost is unknown for every fund holding, so there is no return to compare. Values are shown below.
        </p>
      {/if}
      <p class="muted mt-1 text-xs">
        {#if totals.costUnknown}
          Covers {totals.costKnown} of {totals.holdings} holdings; {totals.costUnknown} without a known cost show their value
          only.
        {:else}
          Covers all {totals.holdings} holdings.
        {/if}
        {#if rateLabel}Deposit: {rateLabel} ({performance.deposit?.basis}).{/if}
      </p>
    </div>

    <div class="-mx-5 mt-4 overflow-x-auto">
      <table class="w-full min-w-[56rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Platform</th>
            <th class="px-5 py-2">Fund</th>
            <th class="px-5 py-2 text-right">Units</th>
            <th class="px-5 py-2 text-right">Value</th>
            <th class="px-5 py-2 text-right">Gain</th>
            <th class="px-5 py-2 text-right">A year</th>
            <th class="px-5 py-2 text-right">Vs deposit</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each performance.holdings as holding, i (i)}
            {@const r = holding.return}
            <tr>
              <td class="px-5 py-2">{holding.platform}</td>
              <td class="px-5 py-2 font-medium">{holding.symbol}</td>
              <td class="tabular px-5 py-2 text-right">{quantity(holding.units)}</td>
              <td class="px-5 py-2 text-right">
                {#if holding.value !== null}<Money value={holding.value} />{:else}<span class="muted">-</span>{/if}
                {#if holding.navDate}<span class="muted block text-xs">NAV {formatDate(holding.navDate)}</span>{/if}
              </td>
              {#if r}
                <td class="px-5 py-2 text-right">
                  <Money value={r.gain} signed />
                  <span class="muted block text-xs"
                    >{formatSignedPercent(r.simpleReturn, 2)} since {formatDate(r.firstDate)}</span
                  >
                </td>
                <td class="tabular px-5 py-2 text-right">{formatPercent(r.annualisedReturn, 2)}</td>
                <td class="px-5 py-2 text-right">
                  {#if r.vsDeposit}
                    <Badge status={VS_DEPOSIT[r.vsDeposit].tone}>{VS_DEPOSIT[r.vsDeposit].label}</Badge>
                    <span class="block text-xs"><Money value={r.differenceVND} signed /></span>
                  {:else}<span class="muted">-</span>{/if}
                </td>
              {:else}
                <td class="px-5 py-2 text-right" colspan="3">
                  <Badge status="neutral">{COST_STATUS_LABEL[holding.costStatus]}</Badge>
                  {#if holding.issues.length}<span class="muted block text-xs">{holding.issues.join('; ')}</span>{/if}
                </td>
              {/if}
            </tr>
          {/each}
        </tbody>
        <tfoot>
          <tr class="border-t border-slate-900/10 font-semibold dark:border-white/10">
            <td class="px-5 py-2" colspan="3">Total</td>
            <td class="px-5 py-2 text-right"><Money value={totals.value} /></td>
            <td class="px-5 py-2 text-right">
              {#if totals.return}<Money value={totals.return.gain} signed />{:else}<span class="muted">-</span>{/if}
            </td>
            <td class="tabular px-5 py-2 text-right">{formatPercent(totals.return?.annualisedReturn, 2)}</td>
            <td class="px-5 py-2 text-right">
              {#if totals.return?.differenceVND != null}<Money value={totals.return.differenceVND} signed />{:else}<span
                  class="muted">-</span
                >{/if}
            </td>
          </tr>
        </tfoot>
      </table>
    </div>
    <p class="muted mt-2 text-xs">
      Each trade is priced at the NAV it was matched at (that day, else the next published). The return a year is
      money-weighted (XIRR); the deposit takes the same cash on the same days.
    </p>
  {/if}
</Card>
