<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { api } from '$lib/api/endpoints';
  import type { MarketPulse, PulseQuote, Timeframe } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon, { type IconName } from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatSignedPercent } from '$lib/format';
  import CorrelationMatrix from '$lib/investments/market/CorrelationMatrix.svelte';
  import SentimentGauge from '$lib/investments/market/SentimentGauge.svelte';
  import SourceBadge from '$lib/investments/market/SourceBadge.svelte';
  import TechnicalsCard from '$lib/investments/market/TechnicalsCard.svelte';
  import TimeframeSwitcher from '$lib/investments/market/TimeframeSwitcher.svelte';
  import {
    REFRESH_INTERVALS,
    REFRESH_STORAGE_KEY,
    TIMEFRAME_LABELS,
    analysisHref,
    changeClass,
    flowTone,
    formatClock,
    formatPrice,
    isRefreshDue,
    momentumLabel,
    parseRefreshInterval,
    parseTimeframe,
    sortQuotes,
    type QuoteSortKey,
    type RefreshInterval,
    type SortDirection,
  } from '$lib/investments/market';

  // WM-INV-11: the engine computes every figure; this page only shows the pulse.
  const timeframe = $derived(parseTimeframe(page.url.searchParams.get('tf')));

  let pulse = $state<MarketPulse | null>(null);
  let loadError = $state('');
  let isLoading = $state(false);
  let lastUpdated = $state<number | null>(null);
  // Drops responses that arrive after a newer request (e.g. a timeframe switch).
  let requestId = 0;

  // Only show a pulse for the selected timeframe, never a stale one from another.
  const current = $derived(pulse?.timeframe === timeframe ? pulse : null);

  async function load(tf: Timeframe, force = false) {
    const id = ++requestId;
    isLoading = true;
    try {
      const result = await api.marketPulse(tf, { force });
      if (id !== requestId) return;
      pulse = result;
      loadError = '';
      lastUpdated = Date.now();
    } catch (error) {
      if (id !== requestId) return;
      loadError = error instanceof Error ? error.message : String(error);
    } finally {
      if (id === requestId) isLoading = false;
    }
  }

  $effect(() => {
    const tf = timeframe;
    untrack(() => load(tf));
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
    if (document.visibilityState !== 'visible' || isLoading) return;
    if (isRefreshDue(lastUpdated, interval, Date.now())) void load(timeframe);
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

  // Asset table sorting; null keeps the engine order.
  let sortKey = $state<QuoteSortKey | null>(null);
  let sortDirection = $state<SortDirection>('desc');
  const assets = $derived(current ? sortQuotes(current.assets, sortKey, sortDirection) : []);
  const failed = $derived(current ? current.assets.filter((quote) => quote.error).length : 0);

  function sortBy(key: QuoteSortKey) {
    if (sortKey === key) {
      if (sortDirection === (key === 'symbol' ? 'asc' : 'desc')) sortDirection = key === 'symbol' ? 'desc' : 'asc';
      else sortKey = null;
    } else {
      sortKey = key;
      sortDirection = key === 'symbol' ? 'asc' : 'desc';
    }
  }

  const ariaSort = (key: QuoteSortKey) =>
    sortKey === key ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none';

  const momentumIcon: Record<PulseQuote['momentum'], IconName> = { fire: 'zap', stable: 'activity', sleep: 'moon' };
  const momentumClass: Record<PulseQuote['momentum'], string> = {
    fire: 'text-orange-500',
    stable: 'text-sky-600 dark:text-sky-400',
    sleep: 'text-slate-400',
  };
  const directionRotation = { up: '', down: 'rotate-180', flat: 'rotate-90' } as const;

  const href = (symbol: string) => `${base}${analysisHref(symbol, timeframe)}`;
</script>

<!-- Sortable asset table header -->
{#snippet sortHeader(key: QuoteSortKey, label: string, align: string)}
  <th class="px-5 py-2 {align}" aria-sort={ariaSort(key)}>
    <button type="button" class="inline-flex items-center gap-1 uppercase" onclick={() => sortBy(key)}>
      {label}
      {#if sortKey === key}<Icon name="arrowUp" size={12} class={sortDirection === 'desc' ? 'rotate-180' : ''} />{/if}
    </button>
  </th>
{/snippet}

<PageHeader title="Market Pulse" subtitle="VN equities, FX, gold and global risk over the selected timeframe">
  {#snippet actions()}
    <TimeframeSwitcher bind:value={() => timeframe, setTimeframe} />
    <label class="glass inline-flex items-center gap-2 px-3 py-1.5 text-sm">
      <Icon name="clock" size={16} class="muted" />
      <span class="sr-only">Auto-refresh</span>
      <select
        class="bg-transparent text-sm focus:outline-none dark:bg-slate-900"
        value={String(interval)}
        onchange={(event) => chooseInterval(event.currentTarget.value)}
        aria-label="Auto-refresh interval"
      >
        {#each REFRESH_INTERVALS as minutes (minutes)}
          <option value={String(minutes)}>{minutes === 0 ? 'Auto-refresh off' : `Every ${minutes} min`}</option>
        {/each}
      </select>
    </label>
    <button type="button" class="btn-ghost" disabled={isLoading} onclick={() => load(timeframe, true)}>
      <Icon name="refresh" size={16} class={isLoading ? 'animate-spin' : ''} /> Refresh
    </button>
  {/snippet}
</PageHeader>

{#if loadError}
  <div class="glass mb-4 border border-rose-500/40 p-3 text-sm text-rose-800 dark:text-rose-200" role="alert">
    <p class="font-medium">Could not load the market pulse{current ? ' - showing the last result' : ''}.</p>
    <p class="mt-1">{loadError}</p>
    {#if !current}
      <button type="button" class="btn-ghost mt-2" onclick={() => load(timeframe, true)}
        ><Icon name="refresh" size={16} /> Retry</button
      >
    {/if}
  </div>
{/if}

{#if current}
  <p class="muted mb-3 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs" aria-live="polite">
    <span>Updated {formatClock(current.generatedAt)} (Vietnam time)</span>
    {#if lastUpdated}<span>· fetched {formatClock(new Date(lastUpdated).toISOString())}</span>{/if}
    {#if isLoading}<span class="inline-flex items-center gap-1"
        ><Icon name="refresh" size={12} class="animate-spin" /> Refreshing</span
      >{/if}
  </p>

  {#if current.topMovers.length}
    <!-- Largest moves over the timeframe -->
    <div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
      {#each current.topMovers as mover (mover.symbol)}
        <a href={href(mover.symbol)} class="glass block p-3 transition hover:ring-2 hover:ring-indigo-500/40">
          <p class="muted text-xs font-medium tracking-wide uppercase">Top mover</p>
          <p class="mt-1 flex items-baseline justify-between gap-2">
            <span class="font-semibold">{mover.symbol}</span>
            <span class="tabular text-sm font-semibold {changeClass(mover.changePct)}"
              >{formatSignedPercent(mover.changePct, 2)}</span
            >
          </p>
          <p class="muted truncate text-xs">{mover.name} · {formatPrice(mover.price)}</p>
        </a>
      {/each}
    </div>
  {/if}

  <Card
    title="Assets"
    subtitle="{TIMEFRAME_LABELS[timeframe]} change is the last bar versus the one before{failed
      ? ` · ${failed} unavailable`
      : ''}"
  >
    {#if assets.length}
      <div class="-mx-5 overflow-x-auto">
        <table class="w-full min-w-[52rem] text-sm">
          <thead>
            <tr class="muted text-left text-xs uppercase">
              {@render sortHeader('symbol', 'Asset', '')}
              <th class="px-5 py-2 text-right">Price</th>
              {@render sortHeader('changePct', TIMEFRAME_LABELS[timeframe], 'text-right')}
              {@render sortHeader('dayChangePct', 'Day', 'text-right')}
              {@render sortHeader('weekChangePct', 'Week', 'text-right')}
              <th class="px-5 py-2 text-center">Trend</th>
              <th class="px-5 py-2">Source</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
            {#each assets as quote (quote.symbol)}
              <tr class={quote.error ? 'opacity-60' : ''}>
                <td class="px-5 py-2">
                  <a href={href(quote.symbol)} class="font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                    >{quote.symbol}</a
                  >
                  <span class="muted block text-xs">{quote.name}</span>
                </td>
                {#if quote.error}
                  <td class="px-5 py-2 text-xs text-rose-700 dark:text-rose-300" colspan="5">
                    <span class="inline-flex items-start gap-1"
                      ><Icon name="alert" size={14} class="mt-0.5" /> Unavailable: {quote.error}</span
                    >
                  </td>
                {:else}
                  <td class="tabular px-5 py-2 text-right font-medium">
                    {formatPrice(quote.price)}
                    {#if quote.currency}<span class="muted text-xs font-normal">{quote.currency}</span>{/if}
                  </td>
                  <td class="tabular px-5 py-2 text-right font-medium {changeClass(quote.changePct)}"
                    >{formatSignedPercent(quote.changePct, 2)}</td
                  >
                  <td class="tabular px-5 py-2 text-right {changeClass(quote.dayChangePct)}"
                    >{formatSignedPercent(quote.dayChangePct, 2)}</td
                  >
                  <td class="tabular px-5 py-2 text-right {changeClass(quote.weekChangePct)}"
                    >{formatSignedPercent(quote.weekChangePct, 2)}</td
                  >
                  <td class="px-5 py-2">
                    <span class="flex items-center justify-center gap-2">
                      <span class={changeClass(quote.changePct)} title="Direction: {quote.direction}">
                        <Icon name="arrowUp" size={16} class={directionRotation[quote.direction]} />
                        <span class="sr-only">{quote.direction}</span>
                      </span>
                      <span class={momentumClass[quote.momentum]} title="Momentum: {momentumLabel[quote.momentum]}">
                        <Icon name={momentumIcon[quote.momentum]} size={16} />
                        <span class="sr-only">{momentumLabel[quote.momentum]}</span>
                      </span>
                    </span>
                  </td>
                {/if}
                <td class="px-5 py-2"><SourceBadge provider={quote.provider} fallback={quote.fallback} /></td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
      <p class="muted mt-3 flex flex-wrap gap-x-4 gap-y-1 text-xs">
        <span class="inline-flex items-center gap-1"
          ><Icon name="zap" size={12} class="text-orange-500" /> Strong move</span
        >
        <span class="inline-flex items-center gap-1"
          ><Icon name="activity" size={12} class="text-sky-600 dark:text-sky-400" /> Steady</span
        >
        <span class="inline-flex items-center gap-1"><Icon name="moon" size={12} class="text-slate-400" /> Quiet</span>
        <span class="inline-flex items-center gap-1"
          ><Icon name="alert" size={12} class="text-amber-600" /> Served by a fallback source (hover for details)</span
        >
      </p>
    {:else}
      <EmptyState title="No assets returned" detail="The engine returned an empty pulse for this timeframe." />
    {/if}
  </Card>

  <div class="mt-4 grid gap-4 lg:grid-cols-2">
    <Card title="Capital flow and scenario" subtitle="Rule-based reading of the moves above">
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
        subtitle="{TIMEFRAME_LABELS[timeframe]} · rule-based heuristics, not a forecast"
        compact
      >
        {#snippet actions()}
          <a class="btn-ghost px-2 text-xs" href={href('VNINDEX')}>Deep dive <Icon name="chevronRight" size={14} /></a>
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
{:else if isLoading}
  <div class="space-y-4" role="status" aria-label="Loading market pulse">
    <div class="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
      {#each Array.from({ length: 5 }, (_, i) => i) as i (i)}
        <div class="glass h-20 animate-pulse"></div>
      {/each}
    </div>
    <div class="glass h-96 animate-pulse"></div>
  </div>
{:else if !loadError}
  <EmptyState title="No market data yet" />
{/if}
