<script lang="ts">
  import type { DashboardSummary } from '$lib/api/types';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { trendDelta, vndToUsd } from '$lib/dashboard';
  import { formatSignedPercent } from '$lib/format';
  import { rateDetail, type LoadedRates } from '$lib/rates';

  // WM-DASH-1/7: net worth with its 30-day trend, in VND and USD. The USD
  // rate loads on its own: null while it is on its way.
  let { summary, usd }: { summary: DashboardSummary; usd: LoadedRates | null } = $props();

  const netWorthUsd = $derived(usd ? vndToUsd(summary.balanceSheet.netWorth, usd.rates.USD) : null);
  const delta30 = $derived(trendDelta(summary.netWorthTrend));
</script>

<Card title="Net worth">
  <div class="text-3xl font-semibold tracking-tight"><Money value={summary.balanceSheet.netWorth} /></div>
  <p class="muted mt-0.5 flex flex-wrap items-center gap-x-2 text-sm">
    {#if !usd}
      <span class="skeleton inline-block h-3.5 w-28" role="status"
        ><span class="sr-only">Loading the USD value…</span></span
      >
    {:else if netWorthUsd !== null}
      {@const quote = usd.quotes.USD}
      <span>≈ <Money value={netWorthUsd} currency="USD" /></span>
      {#if quote}<SourceBadge source={quote.source ?? { provider: quote.provider }} detail={rateDetail(quote)} />{/if}
    {:else}
      <span title={usd.errors.join('; ')}>USD value unavailable</span>
    {/if}
  </p>
  {#if summary.netWorthChangePercent !== null}
    <p class="mt-1 text-sm">
      <span
        class={summary.netWorthChangePercent >= 0
          ? 'text-emerald-600 dark:text-emerald-400'
          : 'text-orange-600 dark:text-orange-400'}>{formatSignedPercent(summary.netWorthChangePercent)}</span
      >
      <span class="muted">vs last month</span>
    </p>
  {/if}
  <div class="mt-4">
    <Sparkline values={summary.netWorthTrend.map((p) => p.value)} width={320} height={64} label="30-day net worth" />
  </div>
  {#if delta30}
    <p class="mt-2 text-sm">
      <span class="muted">30-day change:</span>
      <Money value={delta30.delta} signed tone />
      {#if delta30.percent !== null}<span class="muted">({formatSignedPercent(delta30.percent)})</span>{/if}
    </p>
  {/if}
  <dl class="mt-3 grid grid-cols-[1fr_auto] gap-x-4 gap-y-0.5 text-sm">
    <dt class="muted">Assets{summary.balanceSheet.receivables > 0 ? ', money lent included' : ''}</dt>
    <dd class="text-right"><Money value={summary.balanceSheet.assets} /></dd>
    <dt class="muted">Liabilities{summary.balanceSheet.payables > 0 ? ', money you owe included' : ''}</dt>
    <dd class="text-right"><Money value={summary.balanceSheet.liabilities} /></dd>
  </dl>
</Card>
