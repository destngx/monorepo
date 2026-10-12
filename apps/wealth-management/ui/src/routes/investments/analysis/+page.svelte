<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { MarketAnalysis, Timeframe } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { formatDate, formatPercent, formatSignedPercent } from '$lib/format';
  import SourceBadge from '$lib/investments/market/SourceBadge.svelte';
  import TechnicalsCard from '$lib/investments/market/TechnicalsCard.svelte';
  import TimeframeSwitcher from '$lib/investments/market/TimeframeSwitcher.svelte';
  import {
    PULSE_SYMBOLS,
    TIMEFRAME_LABELS,
    changeClass,
    formatDateTime,
    formatNumber,
    formatPrice,
    groupSeasonality,
    pairTone,
    parseSymbol,
    parseTimeframe,
    rankMark,
    rsiTone,
    trendTone,
  } from '$lib/investments/market';

  // WM-INV-12..16: one ticker's technicals, confluence, seasonality and risk.
  const symbol = $derived(parseSymbol(page.url.searchParams.get('symbol')));
  const timeframe = $derived(parseTimeframe(page.url.searchParams.get('tf')));

  let analysis = $state<MarketAnalysis | null>(null);
  let loadError = $state<{ message: string; status?: number } | null>(null);
  let isLoading = $state(false);
  // Drops responses that arrive after a newer request.
  let requestId = 0;

  // Never show another symbol's or timeframe's figures while the new ones load.
  const current = $derived(analysis?.symbol === symbol && analysis.timeframe === timeframe ? analysis : null);

  async function load(sym: string, tf: Timeframe, force = false) {
    const id = ++requestId;
    if (!sym) {
      analysis = null;
      loadError = null;
      isLoading = false;
      return;
    }
    isLoading = true;
    loadError = null;
    try {
      const result = await api.marketAnalysis(sym, tf, { force });
      if (id !== requestId) return;
      analysis = result;
    } catch (error) {
      if (id !== requestId) return;
      analysis = null;
      loadError = {
        message: error instanceof Error ? error.message : String(error),
        status: error instanceof ApiError ? error.status : undefined,
      };
    } finally {
      if (id === requestId) isLoading = false;
    }
  }

  $effect(() => {
    const [sym, tf] = [symbol, timeframe];
    untrack(() => load(sym, tf));
  });

  function navigate(sym: string, tf: Timeframe) {
    const url = new URL(page.url);
    if (sym) url.searchParams.set('symbol', sym);
    else url.searchParams.delete('symbol');
    url.searchParams.set('tf', tf);
    void goto(url, { keepFocus: true, noScroll: true });
  }

  // The input follows the URL until the user edits it.
  let draft = $state('');
  $effect(() => {
    draft = symbol;
  });

  function submit(event: SubmitEvent) {
    event.preventDefault();
    const next = parseSymbol(draft);
    if (next) navigate(next, timeframe);
  }

  const seasonality = $derived(current ? groupSeasonality(current.seasonality) : []);
  const levels = $derived.by(() => {
    if (!current) return { support: [], resistance: [] };
    const { support, resistance } = current.technicals.supportResistance;
    // Nearest first: support below price descending, resistance above ascending.
    return { support: [...support].sort((a, b) => b - a), resistance: [...resistance].sort((a, b) => a - b) };
  });

  const timeframeOnly = (tf: string) => TIMEFRAME_LABELS[tf as Timeframe] ?? tf;
</script>

<PageHeader
  title="Ticker Analysis"
  subtitle="Technicals, multi-timeframe confluence, seasonality and history-only risk"
>
  {#snippet actions()}
    <TimeframeSwitcher bind:value={() => timeframe, (tf) => navigate(symbol, tf)} />
    <button
      type="button"
      class="btn-ghost"
      disabled={isLoading || !symbol}
      onclick={() => load(symbol, timeframe, true)}
    >
      <Icon name="refresh" size={16} class={isLoading ? 'animate-spin' : ''} /> Refresh
    </button>
  {/snippet}
</PageHeader>

<div class="glass mb-4 p-4">
  <form class="flex flex-wrap items-end gap-2" onsubmit={submit}>
    <div class="min-w-48 flex-1 sm:max-w-xs">
      <label class="label" for="ticker-symbol">Symbol</label>
      <input
        id="ticker-symbol"
        class="field uppercase"
        placeholder="e.g. FPT, VNINDEX, USDVND"
        autocomplete="off"
        spellcheck="false"
        maxlength="20"
        bind:value={() => draft, (value) => (draft = value.toUpperCase())}
      />
    </div>
    <button type="submit" class="btn-primary" disabled={!parseSymbol(draft)}
      ><Icon name="search" size={16} /> Analyse</button
    >
  </form>
  <div class="mt-3 flex flex-wrap gap-1.5" role="group" aria-label="Quick picks">
    {#each PULSE_SYMBOLS as pick (pick)}
      <button
        type="button"
        class="rounded-full px-2.5 py-0.5 text-xs font-medium transition {pick === symbol
          ? 'bg-indigo-600 text-white'
          : 'bg-slate-900/5 text-slate-700 hover:bg-slate-900/10 dark:bg-white/10 dark:text-slate-300 dark:hover:bg-white/15'}"
        aria-pressed={pick === symbol}
        onclick={() => navigate(pick, timeframe)}>{pick}</button
      >
    {/each}
  </div>
</div>

{#if !symbol}
  <div class="glass">
    <EmptyState title="Pick a ticker" detail="Enter a symbol or choose one of the market pulse instruments above." />
  </div>
{:else if loadError}
  <div class="glass border border-rose-500/40 p-4 text-sm text-rose-800 dark:text-rose-200" role="alert">
    <p class="font-medium">
      Could not analyse {symbol} on {TIMEFRAME_LABELS[timeframe]}{loadError.status
        ? ` (HTTP ${loadError.status})`
        : ''}.
    </p>
    <p class="mt-1">{loadError.message}</p>
    {#if symbol === 'SJC' && (timeframe === '1h' || timeframe === '4h')}
      <p class="mt-1">SJC gold is only published daily; try the 1D or 1W timeframe.</p>
    {/if}
    <button type="button" class="btn-ghost mt-2" onclick={() => load(symbol, timeframe, true)}
      ><Icon name="refresh" size={16} /> Retry</button
    >
  </div>
{:else if current}
  {@const technicals = current.technicals}
  <!-- Header -->
  <div class="glass mb-4 flex flex-wrap items-end justify-between gap-4 p-5">
    <div>
      <p class="muted text-xs font-medium tracking-wide uppercase">{TIMEFRAME_LABELS[current.timeframe]} view</p>
      <h2 class="mt-1 text-2xl font-semibold tracking-tight">{current.symbol}</h2>
    </div>
    <div class="text-right">
      <p class="tabular text-2xl font-semibold">
        {formatPrice(technicals.price)}
        {#if current.currency}<span class="muted text-sm font-normal">{current.currency}</span>{/if}
      </p>
      <p class="muted mt-1 flex flex-wrap items-center justify-end gap-2 text-xs">
        <SourceBadge provider={current.provider} fallback={current.fallback} />
        <span>Generated {formatDateTime(current.generatedAt)}</span>
        {#if isLoading}<Icon name="refresh" size={12} class="animate-spin" />{/if}
      </p>
      {#if current.fallback}
        <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">Fallback: {current.fallback}</p>
      {/if}
    </div>
  </div>

  <div class="grid gap-4 lg:grid-cols-3">
    <!-- WM-INV-12 -->
    <TechnicalsCard {technicals} class="lg:col-span-2" />

    <div class="space-y-4">
      <Card title="Support and resistance" subtitle="Nearest levels first">
        <div class="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p class="mb-1 text-xs font-semibold text-rose-700 uppercase dark:text-rose-300">Resistance</p>
            {#if levels.resistance.length}
              <ul class="tabular space-y-0.5">
                {#each levels.resistance as level, i (i)}<li>{formatPrice(level)}</li>{/each}
              </ul>
            {:else}<p class="muted text-xs">None found</p>{/if}
          </div>
          <div>
            <p class="mb-1 text-xs font-semibold text-emerald-700 uppercase dark:text-emerald-300">Support</p>
            {#if levels.support.length}
              <ul class="tabular space-y-0.5">
                {#each levels.support as level, i (i)}<li>{formatPrice(level)}</li>{/each}
              </ul>
            {:else}<p class="muted text-xs">None found</p>{/if}
          </div>
        </div>
      </Card>

      <Card title="Open fair value gaps" subtitle="Price imbalances not yet filled">
        {#if technicals.fairValueGaps.length}
          <ul class="space-y-1.5 text-sm">
            {#each technicals.fairValueGaps as gap, i (i)}
              <li class="flex flex-wrap items-center justify-between gap-2">
                <Badge tone={gap.type === 'BULLISH' ? 'green' : 'red'}>{gap.type}</Badge>
                <span class="tabular">{formatPrice(gap.bottom)} - {formatPrice(gap.top)}</span>
                <span class="muted text-xs">{formatDate(gap.time)}</span>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="muted text-sm">No open gaps.</p>
        {/if}
      </Card>
    </div>
  </div>

  <!-- WM-INV-13 -->
  <Card
    title="Multi-timeframe confluence"
    subtitle="How the timeframes agree; higher timeframes weigh more"
    class="mt-4"
  >
    {#snippet actions()}
      <Badge tone={trendTone[current.confluence.bias]}>Bias: {current.confluence.bias}</Badge>
      <Badge tone="blue">Score {current.confluence.score}/10</Badge>
    {/snippet}
    <div class="-mx-5 overflow-x-auto">
      <table class="w-full min-w-[40rem] text-sm">
        <thead>
          <tr class="muted text-left text-xs uppercase">
            <th class="px-5 py-2">Timeframe</th>
            <th class="px-5 py-2">Trend</th>
            <th class="px-5 py-2">Phase</th>
            <th class="px-5 py-2 text-right">RSI</th>
            <th class="px-5 py-2 text-right">Price</th>
            <th class="px-5 py-2">Source</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
          {#each current.confluence.views as view (view.timeframe)}
            <tr class={view.error ? 'opacity-60' : view.timeframe === current.timeframe ? 'bg-indigo-500/5' : ''}>
              <td class="px-5 py-2 font-medium">{timeframeOnly(view.timeframe)}</td>
              {#if view.error}
                <td class="px-5 py-2 text-xs text-rose-700 dark:text-rose-300" colspan="4">
                  <span class="inline-flex items-start gap-1"
                    ><Icon name="alert" size={14} class="mt-0.5" /> {view.error}</span
                  >
                </td>
              {:else}
                <td class="px-5 py-2"><Badge tone={trendTone[view.trend]}>{view.trend}</Badge></td>
                <td class="px-5 py-2">{view.phase}</td>
                <td class="tabular px-5 py-2 text-right">
                  {#if view.rsi !== null}
                    <span
                      class={rsiTone(view.rsi) === 'red'
                        ? 'text-rose-600 dark:text-rose-400'
                        : rsiTone(view.rsi) === 'green'
                          ? 'text-emerald-600 dark:text-emerald-400'
                          : ''}>{formatNumber(view.rsi, 1)}</span
                    >
                  {:else}<span class="muted">-</span>{/if}
                </td>
                <td class="tabular px-5 py-2 text-right">{formatPrice(view.price)}</td>
              {/if}
              <td class="px-5 py-2"><SourceBadge provider={view.provider} /></td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
    {#if current.confluence.pairs.length}
      <div class="mt-3 flex flex-wrap gap-2 text-sm">
        {#each current.confluence.pairs as pair (`${pair.higher}-${pair.lower}`)}
          <span class="inline-flex items-center gap-1.5">
            <span class="muted text-xs">{timeframeOnly(pair.higher)} vs {timeframeOnly(pair.lower)}</span>
            <Badge tone={pairTone[pair.status]}>{pair.status}</Badge>
          </span>
        {/each}
      </div>
    {/if}
  </Card>

  <!-- WM-INV-14 -->
  <Card title="Seasonality" subtitle="Daily returns grouped by calendar bucket; rank 1 is the best" class="mt-4">
    {#if seasonality.length}
      <div class="grid gap-6 xl:grid-cols-3">
        {#each seasonality as group (group.type)}
          <div class="min-w-0">
            <h3 class="mb-2 text-xs font-semibold tracking-wide text-slate-600 uppercase dark:text-slate-400">
              {group.title}
            </h3>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[26rem] text-xs">
                <thead>
                  <tr class="muted text-left uppercase">
                    <th class="py-1.5 pr-2">Bucket</th>
                    <th class="px-2 py-1.5 text-right">Avg</th>
                    <th class="px-2 py-1.5 text-right">Win</th>
                    <th class="px-2 py-1.5 text-right" title="Profit factor">PF</th>
                    <th class="px-2 py-1.5 text-right" title="Standard deviation">SD</th>
                    <th class="px-2 py-1.5 text-right">Score</th>
                    <th class="px-2 py-1.5 text-right">N</th>
                    <th class="py-1.5 pl-2 text-right">Rank</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
                  {#each group.rows as row (row.key)}
                    {@const mark = rankMark(row.rank, group.worstRank)}
                    <tr class={mark === 'best' ? 'bg-emerald-500/10' : mark === 'worst' ? 'bg-rose-500/10' : ''}>
                      <td class="py-1.5 pr-2 font-medium">{row.label}</td>
                      <td class="tabular px-2 py-1.5 text-right {changeClass(row.avgReturn)}"
                        >{formatSignedPercent(row.avgReturn, 2)}</td
                      >
                      <td class="tabular px-2 py-1.5 text-right">{formatPercent(row.winRate, 0)}</td>
                      <td class="tabular px-2 py-1.5 text-right">{formatNumber(row.profitFactor)}</td>
                      <td class="tabular px-2 py-1.5 text-right">{formatPercent(row.stdDev, 2)}</td>
                      <td class="tabular px-2 py-1.5 text-right">{formatNumber(row.score)}</td>
                      <td class="tabular px-2 py-1.5 text-right">{row.n}</td>
                      <td class="tabular py-1.5 pl-2 text-right">
                        {#if mark === 'best'}<Badge tone="green">Best</Badge>{:else if mark === 'worst'}<Badge
                            tone="red">Worst</Badge
                          >{:else}{row.rank}{/if}
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <p class="muted text-sm">Not enough daily history to group returns.</p>
    {/if}
  </Card>

  <!-- WM-INV-16: history-only risk, labelled as a heuristic -->
  <Card title="Risk" subtitle="From daily history" class="mt-4">
    {#snippet actions()}
      <Badge tone="amber">Heuristic - history only, not a forecast</Badge>
    {/snippet}
    {#if current.risk}
      {@const risk = current.risk}
      <dl class="grid grid-cols-2 gap-x-6 gap-y-3 text-sm sm:grid-cols-3 lg:grid-cols-5">
        <div>
          <dt class="muted text-xs">Period return</dt>
          <dd class="tabular font-medium {changeClass(risk.periodReturn)}">
            {formatSignedPercent(risk.periodReturn, 2)}
          </dd>
          <dd class="muted text-xs">{formatDate(risk.from)} - {formatDate(risk.to)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Annualised return</dt>
          <dd class="tabular font-medium {changeClass(risk.annualisedReturn)}">
            {formatSignedPercent(risk.annualisedReturn, 2)}
          </dd>
        </div>
        <div>
          <dt class="muted text-xs">Annualised volatility</dt>
          <dd class="tabular font-medium">{formatPercent(risk.annualisedVolatility, 2)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Max drawdown</dt>
          <dd class="tabular font-medium text-rose-600 dark:text-rose-400">{formatPercent(risk.maxDrawdown, 2)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Current drawdown</dt>
          <dd class="tabular font-medium text-rose-600 dark:text-rose-400">
            {formatPercent(risk.currentDrawdown, 2)}
          </dd>
        </div>
      </dl>

      {#if risk.simulation}
        {@const sim = risk.simulation}
        <div class="mt-4 rounded-xl border border-amber-500/30 bg-amber-500/5 p-3">
          <p class="text-xs font-semibold tracking-wide text-amber-800 uppercase dark:text-amber-200">
            Bootstrap resample of past daily returns over {sim.horizonDays} trading days ({sim.paths.toLocaleString(
              'en',
            )} paths)
          </p>
          <dl class="mt-2 grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <dt class="muted text-xs">P10 price</dt>
              <dd class="tabular font-medium">{formatPrice(sim.p10)}</dd>
            </div>
            <div>
              <dt class="muted text-xs">P50 price</dt>
              <dd class="tabular font-medium">{formatPrice(sim.p50)}</dd>
            </div>
            <div>
              <dt class="muted text-xs">P90 price</dt>
              <dd class="tabular font-medium">{formatPrice(sim.p90)}</dd>
            </div>
            <div>
              <dt class="muted text-xs">Paths ending lower</dt>
              <dd class="tabular font-medium">{formatPercent(sim.probabilityOfLoss, 1)}</dd>
            </div>
          </dl>
        </div>
      {:else}
        <p class="muted mt-4 text-sm">Not enough history to resample a simulation.</p>
      {/if}
      <p class="muted mt-3 text-xs"><span class="font-medium">Method:</span> {risk.method}</p>
    {:else}
      <p class="muted text-sm">Risk statistics need daily history, which is unavailable for this symbol.</p>
    {/if}
  </Card>

  {@const notes = [...technicals.notes, ...current.notes]}
  {#if notes.length}
    <Card title="Notes" class="mt-4">
      <ul class="list-disc space-y-1 pl-5 text-sm">
        {#each notes as note, i (i)}<li>{note}</li>{/each}
      </ul>
    </Card>
  {/if}

  <!-- WM-AI-14: on demand only; the model reads the engine figures above -->
  <InsightCard
    title="AI insight"
    subtitle="{current.symbol} on {TIMEFRAME_LABELS[current.timeframe]} - commentary on the figures above"
    hint="AI commentary on the market figures above, with recent web news when search is available."
    class="mt-4"
    load={(force, signal) => api.tickerInsight(symbol, timeframe, { force, signal })}
  />
{:else}
  <div class="space-y-4" role="status" aria-label="Loading analysis">
    <div class="glass h-24 animate-pulse"></div>
    <div class="grid gap-4 lg:grid-cols-3">
      <div class="glass h-72 animate-pulse lg:col-span-2"></div>
      <div class="glass h-72 animate-pulse"></div>
    </div>
    <div class="glass h-48 animate-pulse"></div>
  </div>
{/if}
