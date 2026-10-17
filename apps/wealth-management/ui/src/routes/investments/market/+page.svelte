<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import { defer, type Streamed } from '$lib/api/load';
  import type { MarketPulse, Timeframe } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import SkeletonTable from '$lib/components/skeleton/SkeletonTable.svelte';
  import AssetsTable from '$lib/investments/market/AssetsTable.svelte';
  import CorrelationMatrix from '$lib/investments/market/CorrelationMatrix.svelte';
  import SentimentGauge from '$lib/investments/market/SentimentGauge.svelte';
  import { symbolHref } from '$lib/investments/market/link';
  import RuleBasedNote from '$lib/investments/market/RuleBasedNote.svelte';
  import TechnicalsCard from '$lib/investments/market/TechnicalsCard.svelte';
  import TimeframeSwitcher from '$lib/investments/market/TimeframeSwitcher.svelte';
  import TopMovers from '$lib/investments/market/TopMovers.svelte';
  import {
    REFRESH_INTERVALS,
    REFRESH_STORAGE_KEY,
    TIMEFRAME_LABELS,
    flowTone,
    formatClock,
    isRefreshDue,
    parseRefreshInterval,
    parseTimeframe,
    type RefreshInterval,
  } from '$lib/investments/market';

  // WM-INV-11: the engine computes every figure; this page only shows the pulse.
  // The header (timeframe, auto-refresh, Refresh) shows at once and the pulse
  // streams in below it (WM-UI-16). A refresh keeps the shown pulse with a
  // Refreshing badge; another timeframe shows the skeleton, never a stale one.
  const timeframe = $derived(parseTimeframe(page.url.searchParams.get('tf')));

  // Bumped by Refresh, Retry and auto-refresh; the next request skips the
  // engine cache when `forceNext` is set.
  let reloads = $state(0);
  let forceNext = false;
  const pulseData = $derived.by<Streamed<MarketPulse>>(() => {
    const tf = timeframe;
    void reloads;
    const force = forceNext;
    forceNext = false;
    return defer(() => api.marketPulse(tf, { force }));
  });
  const pulse = track(() => pulseData, { scope: () => timeframe });

  function reload(force: boolean): Streamed<MarketPulse> {
    forceNext = force;
    reloads++;
    return pulseData;
  }

  // Busy until the pulse answers; a failure toasts and shows under the old pulse.
  async function refresh() {
    const result = await reload(true);
    if (!result.ok) throw new Error(result.error);
  }

  let lastUpdated = $state<number | null>(null);
  $effect(() => {
    const request = pulseData;
    void request.then((result) => {
      if (result.ok && untrack(() => pulseData) === request) lastUpdated = Date.now();
    });
  });

  function setTimeframe(tf: Timeframe) {
    const url = new URL(page.url);
    url.searchParams.set('tf', tf);
    void goto(url, { keepFocus: true, noScroll: true, replaceState: true });
  }

  // Auto-refresh (persisted); ticks are skipped while the tab is hidden and a
  // missed refresh runs as soon as it is visible again.
  let interval = $state<RefreshInterval>(parseRefreshInterval(globalThis.localStorage?.getItem(REFRESH_STORAGE_KEY)));

  function chooseInterval(value: string) {
    interval = parseRefreshInterval(value);
    if (interval === 0) globalThis.localStorage?.removeItem(REFRESH_STORAGE_KEY);
    else globalThis.localStorage?.setItem(REFRESH_STORAGE_KEY, String(interval));
  }

  function refreshIfDue() {
    if (document.visibilityState !== 'visible' || pulse.pending) return;
    if (isRefreshDue(lastUpdated, interval, Date.now())) reload(false);
  }

  $effect(() => {
    if (interval === 0) return;
    // Check every 15 s so a visible tab refreshes close to the chosen interval.
    const timer = setInterval(() => untrack(refreshIfDue), 15_000);
    const onVisible = () => untrack(refreshIfDue);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  });

  const href = (symbol: string) => symbolHref(symbol, timeframe);
</script>

<PageHeader title="Market Pulse" subtitle="VN equities, FX, gold and global risk over the selected timeframe">
  {#snippet actions()}
    <TimeframeSwitcher bind:value={() => timeframe, setTimeframe} />
    <label
      class="glass inline-flex items-center gap-2 px-3 py-1.5 text-sm transition-colors focus-within:ring-2 focus-within:ring-indigo-500/40 hover:border-indigo-400/40"
    >
      <Icon name="clock" size={16} class="muted" />
      <span class="sr-only">Auto-refresh</span>
      <select
        class="cursor-pointer bg-transparent text-sm focus:outline-none dark:bg-slate-900"
        value={String(interval)}
        onchange={(event) => chooseInterval(event.currentTarget.value)}
        aria-label="Auto-refresh interval"
      >
        {#each REFRESH_INTERVALS as minutes (minutes)}
          <option value={String(minutes)}>{minutes === 0 ? 'Auto-refresh off' : `Every ${minutes} min`}</option>
        {/each}
      </select>
    </label>
    <Button
      icon="refresh"
      busy={pulse.pending && !pulse.loading}
      busyText="Refreshing…"
      success="Market pulse refreshed."
      errorContext="Could not refresh the market pulse"
      onclick={refresh}>Refresh</Button
    >
  {/snippet}
</PageHeader>

<Deferred data={pulseData} scope={timeframe} label="the market pulse" retry={() => reload(true)}>
  {#snippet children(current)}
    <div>
      <p class="muted mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs" aria-live="polite">
        <span>Updated {formatClock(current.generatedAt)} (Vietnam time)</span>
        {#if lastUpdated}<span>· fetched {formatClock(new Date(lastUpdated).toISOString())}</span>{/if}
      </p>
      <RuleBasedNote class="mb-3" />

      {#if current.topMovers.length}
        <TopMovers movers={current.topMovers} {timeframe} />
      {/if}

      <AssetsTable quotes={current.assets} {timeframe} />

      <div class="mt-4 grid gap-4 lg:grid-cols-2">
        <Card title="Capital flow and scenario" subtitle="What the moves above suggest">
          <div class="flex flex-wrap items-center gap-2">
            <Badge tone={flowTone[current.capitalFlow.signal]}>{current.capitalFlow.signal}</Badge>
            <span class="text-sm">{current.capitalFlow.summary}</span>
          </div>
          <div class="mt-4 rounded-xl border border-slate-900/10 p-3 dark:border-white/10">
            <p class="flex flex-wrap items-center gap-2">
              <span class="font-semibold">{current.scenario.name}</span>
              <Badge tone="violet">{current.scenario.regime}</Badge>
            </p>
            <p class="mt-2 text-sm">{current.scenario.summary}</p>
            <p class="mt-2 flex gap-2 text-sm font-medium">
              <Icon name="check" size={16} class="mt-0.5 text-emerald-600" />{current.scenario.action}
            </p>
          </div>
        </Card>

        <Card title="Sentiment" subtitle="Fear / greed gauge from VIX and VN-Index RSI">
          {#if current.sentiment}
            <SentimentGauge score={current.sentiment.score} label={current.sentiment.label} />
          {:else}
            <p class="muted text-sm">Sentiment needs both VIX and the VN-Index RSI; one is unavailable right now.</p>
          {/if}
        </Card>
      </div>

      <div class="mt-4 grid gap-4 lg:grid-cols-2">
        {#if current.index}
          <TechnicalsCard
            technicals={current.index}
            title="VN-Index technicals"
            subtitle={TIMEFRAME_LABELS[timeframe]}
            compact
          >
            {#snippet actions()}
              <a class="btn-ghost px-2 text-xs" href={href('VNINDEX')}
                >Deep dive <Icon name="chevronRight" size={14} /></a
              >
            {/snippet}
          </TechnicalsCard>
        {:else}
          <Card title="VN-Index technicals">
            <p class="muted text-sm">VN-Index technicals are unavailable for this timeframe.</p>
          </Card>
        {/if}

        <Card title="Correlation" subtitle="Pearson correlation of {TIMEFRAME_LABELS[timeframe]} returns">
          <CorrelationMatrix matrix={current.correlation} />
        </Card>
      </div>

      {#if current.notes.length}
        <Card title="Notes" class="mt-4">
          <ul class="list-disc space-y-1 pl-5 text-sm">
            {#each current.notes as note, i (i)}<li>{note}</li>{/each}
          </ul>
        </Card>
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
        {#each { length: 5 }, i (i)}<SkeletonStat label="top mover {i + 1}" />{/each}
      </div>
      <section class="glass p-5" aria-label="Assets">
        <h2 class="mb-4 text-sm font-semibold tracking-wide text-slate-700 dark:text-slate-200">Assets</h2>
        <SkeletonTable rows={8} columns={6} label="asset quotes" />
      </section>
      <div class="grid gap-4 lg:grid-cols-2">
        <SkeletonCard title="Capital flow and scenario" lines={3} />
        <SkeletonCard title="Sentiment" lines={2} chart={80} />
      </div>
    </div>
  {/snippet}
</Deferred>
