<script lang="ts">
  import { base } from '$app/paths';
  import { track, Card, Deferred, EmptyState, Icon, SkeletonCard, SkeletonChart, SkeletonStat } from '@libs/ui-kit';
  import Donut from '$lib/charts/Donut.svelte';
  import { PALETTE } from '$lib/charts/palette';
  import Badge from '$lib/components/Badge.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent, formatSignedPercent, formatVND, todayISO } from '$lib/format';
  import {
    downsample,
    FUND_TYPES,
    periodReturn,
    sliceRange,
    type FundDetail,
    type Range,
  } from '$lib/investments/fmarket';
  import MultiLineChart from '$lib/investments/fmarket/MultiLineChart.svelte';
  import RangeButtons from '$lib/investments/fmarket/RangeButtons.svelte';

  // The header (code, back link) shows at once; the fund detail and its NAV
  // history stream in below it (WM-UI-16), each with its skeleton and error.
  let { data } = $props();
  const detail = track(() => data.detail, { scope: () => data.code });
  const nav = track(() => data.nav, { scope: () => data.code });
  const fund = $derived(detail.value?.fund);
  // Back to the list of this fund's type; stock is the list's default.
  const backQuery = $derived(
    fund?.assetCode && fund.assetCode !== 'STOCK' && FUND_TYPES.some((t) => t.key === fund.assetCode)
      ? `?type=${fund.assetCode}`
      : '',
  );

  // WM-INV-10 fund detail: NAV chart by range.
  let range = $state<Range>('1Y');
  const navPoints = $derived(nav.value ? sliceRange(nav.value.data, range, todayISO()) : []);
  const chartPoints = $derived(downsample(navPoints));
  const rangeReturn = $derived(periodReturn(navPoints));

  const periodLabels: [string, string][] = [
    ['navTo1Months', '1M'],
    ['navTo3Months', '3M'],
    ['navTo6Months', '6M'],
    ['navToLastYear', 'YTD'],
    ['navTo12Months', '1Y'],
    ['navTo24Months', '2Y'],
    ['navTo36Months', '3Y'],
    ['navTo60Months', '5Y'],
    ['navTo10Years', '10Y'],
    ['navToBeginning', 'Inception'],
  ];
  const periodsOf = (value: FundDetail) =>
    periodLabels
      .map(([key, label]) => ({ label, value: value.changes[key] ?? null }))
      .filter((period) => period.value !== null);

  const feeTypes: Record<string, string> = { BUY: 'Subscription', SELL: 'Redemption', TRANSFER: 'Switch' };
  const feeGroupsOf = (value: FundDetail) =>
    [...new Set(value.fees.map((fee) => fee.type))].map((type) => ({
      type,
      label: feeTypes[type] ?? type,
      rows: value.fees.filter((fee) => fee.type === type),
    }));

  const navFormat = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 2 });
  const units = (value: number | null) => (value === null ? '-' : `${navFormat.format(value)} units`);
  const tone = (value: number | null) =>
    value === null
      ? 'muted'
      : value >= 0
        ? 'text-emerald-600 dark:text-emerald-400'
        : 'text-orange-600 dark:text-orange-400';
</script>

<PageHeader title={fund?.code ?? data.code} subtitle={fund?.name}>
  {#snippet actions()}
    <a href="{base}/investments/funds{backQuery}" class="btn-ghost"><Icon name="chevronLeft" size={16} /> All funds</a>
  {/snippet}
</PageHeader>

<Deferred data={data.detail} scope={data.code} label="fund {data.code}">
  {#snippet children({ fund, source })}
    {@const periods = periodsOf(fund)}
    {@const feeGroups = feeGroupsOf(fund)}
    {@const showProgram = new Set(fund.fees.map((fee) => fee.program)).size > 1}
    <div class="mb-4 flex flex-wrap gap-2">
      {#if fund.issuer}<Badge tone="violet">{fund.issuer}</Badge>{/if}
      {#if fund.assetType}<Badge tone="blue">{fund.assetType}</Badge>{/if}
      {#if fund.riskLevel}<Badge tone="amber">Risk: {fund.riskLevel}</Badge>{/if}
      {#if source}<SourceBadge {source} />{/if}
    </div>

    <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
      <Stat label="NAV per unit" hint={fund.updatedAt ? `Updated ${formatDate(fund.updatedAt)}` : undefined}>
        <span class="tabular">{fund.nav === null ? '-' : formatVND(fund.nav)}</span>
      </Stat>
      <Stat label="12-month return">
        <span class="tabular {tone(fund.return12m)}">{formatSignedPercent(fund.return12m, 2)}</span>
      </Stat>
      <Stat label="3Y annualised">
        <span class="tabular {tone(fund.annualized36m)}">{formatSignedPercent(fund.annualized36m, 2)}</span>
      </Stat>
      <Stat label="Since inception" hint={fund.inceptionDate ? `From ${formatDate(fund.inceptionDate)}` : undefined}>
        <span class="tabular {tone(fund.changes.navToBeginning ?? null)}"
          >{formatSignedPercent(fund.changes.navToBeginning, 2)}</span
        >
      </Stat>
    </div>

    <Card title="NAV history" subtitle="₫ per unit" class="mt-4">
      {#snippet actions()}
        {#if nav.value?.source}<SourceBadge source={nav.value.source} />{/if}
        <RangeButtons value={range} onchange={(next) => (range = next)} />
      {/snippet}
      <Deferred data={data.nav} scope={data.code} label="the NAV history">
        {#snippet children()}
          <div>
            <MultiLineChart
              label="{fund.code} NAV history"
              dates={chartPoints.map((point) => point.date)}
              series={[{ name: 'NAV', color: PALETTE[0], values: chartPoints.map((point) => point.value) }]}
            />
            {#if rangeReturn !== null}
              <p class="muted mt-2 text-xs">
                Change over {range}:
                <span class="font-medium {tone(rangeReturn)}">{formatSignedPercent(rangeReturn, 2)}</span>
              </p>
            {/if}
          </div>
        {/snippet}
        {#snippet skeleton()}<SkeletonChart height={220} label="the NAV history" />{/snippet}
      </Deferred>
      {#if periods.length}
        <div class="mt-4 flex flex-wrap gap-2">
          {#each periods as period (period.label)}
            <span class="rounded-xl bg-stone-900/5 px-2.5 py-1 text-xs dark:bg-white/5">
              <span class="muted">{period.label}</span>
              <span class="tabular font-medium {tone(period.value)}">{formatSignedPercent(period.value, 2)}</span>
            </span>
          {/each}
        </div>
      {/if}
    </Card>

    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card title="Overview">
        {#if fund.description}<p class="mb-4 text-sm leading-relaxed">{fund.description}</p>{/if}
        <dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          <dt class="muted">Management company</dt>
          <dd>{fund.issuer ?? '-'}</dd>
          <dt class="muted">Inception</dt>
          <dd>{formatDate(fund.inceptionDate)}</dd>
          <dt class="muted">Management fee</dt>
          <dd>{formatPercent(fund.managementFee, 2)} / year</dd>
          {#if fund.performanceFee !== null}
            <dt class="muted">Performance fee</dt>
            <dd>{formatPercent(fund.performanceFee, 2)}</dd>
          {/if}
          <dt class="muted">Order cut-off</dt>
          <dd>{fund.orderCutoff ?? '-'}</dd>
          <dt class="muted">Next trading day</dt>
          <dd>{fund.nextTradingDate ?? '-'}</dd>
          {#if fund.website}
            <dt class="muted">Website</dt>
            <dd class="truncate">
              <a href={fund.website} target="_blank" rel="noopener noreferrer" class="link"
                >{new URL(fund.website).hostname}</a
              >
            </dd>
          {/if}
        </dl>
      </Card>

      <Card title="Minimums">
        <dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          <dt class="muted">Minimum purchase</dt>
          <dd class="tabular">{formatVND(fund.buyMinValue)}</dd>
          <dt class="muted">Minimum redemption</dt>
          <dd class="tabular">{units(fund.sellMinUnits)}</dd>
          <dt class="muted">Minimum holding</dt>
          <dd class="tabular">{units(fund.holdingMinUnits)}</dd>
        </dl>
      </Card>
    </div>

    <Card title="Fees" subtitle="Percent of the transaction value" class="mt-4">
      {#if feeGroups.length}
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-3">
          {#each feeGroups as group (group.type)}
            <div>
              <h3 class="mb-2 text-xs font-semibold tracking-wide uppercase">{group.label}</h3>
              <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5">
                {#each group.rows as fee, i (i)}
                  <li class="flex justify-between gap-3 py-1.5">
                    <span>
                      {fee.condition}
                      {#if fee.targets.length}<span class="muted block text-xs">To {fee.targets.join(', ')}</span>{/if}
                      {#if showProgram && fee.program}<span class="muted block text-xs">{fee.program}</span>{/if}
                    </span>
                    <span class="tabular font-medium">{formatPercent(fee.fee, 2)}</span>
                  </li>
                {/each}
              </ul>
            </div>
          {/each}
        </div>
      {:else}
        <EmptyState title="No fee schedule published" />
      {/if}
    </Card>

    <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card title="Top 10 holdings" subtitle={fund.holdingsAsOf ? `As of ${fund.holdingsAsOf}` : undefined}>
        {#if fund.topHoldings.length}
          <ul class="space-y-2 text-sm">
            {#each fund.topHoldings as holding, i (i)}
              <li class="grid grid-cols-[4.5rem_1fr_3.5rem] items-center gap-3">
                <span class="font-medium">{holding.code}</span>
                <span class="min-w-0">
                  <span class="block h-2 overflow-hidden rounded-sm bg-stone-900/5 dark:bg-white/5">
                    <span
                      class="block h-full rounded-sm bg-accent-500/80"
                      style:width="{Math.min((holding.percent / fund.topHoldings[0].percent) * 100, 100)}%"
                    ></span>
                  </span>
                  {#if holding.industry}<span class="muted block truncate text-xs">{holding.industry}</span>{/if}
                </span>
                <span class="tabular text-right">{formatPercent(holding.percent, 2)}</span>
              </li>
            {/each}
          </ul>
        {:else}
          <EmptyState title="No holdings published" />
        {/if}
      </Card>

      <Card title="Asset allocation" subtitle={fund.holdingsAsOf ? `As of ${fund.holdingsAsOf}` : undefined}>
        {#if fund.allocation.length}
          <div class="flex flex-col items-center gap-6 sm:flex-row">
            <Donut
              label="Asset allocation"
              size={160}
              items={fund.allocation.map((share) => ({ label: share.label, value: share.percent, color: share.color }))}
            />
            <ul class="w-full space-y-1.5 text-sm">
              {#each fund.allocation as share, i (i)}
                <li class="flex items-center justify-between gap-3">
                  <span class="inline-flex items-center gap-2"
                    ><span class="size-2.5 rounded-sm" style:background={share.color ?? PALETTE[i % PALETTE.length]}
                    ></span>{share.label}</span
                  >
                  <span class="tabular font-medium">{formatPercent(share.percent, 2)}</span>
                </li>
              {/each}
            </ul>
          </div>
        {:else}
          <EmptyState title="No allocation published" />
        {/if}
        {#if fund.industries.length}
          <h3 class="mt-6 mb-2 text-xs font-semibold tracking-wide uppercase">By industry</h3>
          <ul class="grid grid-cols-1 gap-x-6 gap-y-1 text-sm sm:grid-cols-2">
            {#each fund.industries as share, i (i)}
              <li class="flex justify-between gap-3 border-b border-stone-900/5 py-1 dark:border-white/5">
                <span class="truncate">{share.label}</span>
                <span class="tabular">{formatPercent(share.percent, 2)}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </Card>
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {#each ['NAV per unit', '12-month return', '3Y annualised return', 'return since inception'] as label (label)}
          <SkeletonStat {label} />
        {/each}
      </div>
      <SkeletonCard title="NAV history" lines={1} chart={220} class="mt-4" />
      <div class="mt-4 grid grid-cols-1 gap-4 lg:grid-cols-2">
        <SkeletonCard title="Overview" lines={5} />
        <SkeletonCard title="Minimums" lines={3} />
      </div>
    </div>
  {/snippet}
</Deferred>
