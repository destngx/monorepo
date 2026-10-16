<script lang="ts">
  import { untrack } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { MarketAnalysis, Timeframe } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ConfluenceCard from '$lib/investments/market/ConfluenceCard.svelte';
  import LevelsPanel from '$lib/investments/market/LevelsPanel.svelte';
  import RiskCard from '$lib/investments/market/RiskCard.svelte';
  import RuleBasedNote from '$lib/investments/market/RuleBasedNote.svelte';
  import SeasonalityCard from '$lib/investments/market/SeasonalityCard.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import TechnicalsCard from '$lib/investments/market/TechnicalsCard.svelte';
  import TimeframeSwitcher from '$lib/investments/market/TimeframeSwitcher.svelte';
  import {
    PULSE_SYMBOLS,
    TIMEFRAME_LABELS,
    formatDateTime,
    formatPrice,
    parseSymbol,
    parseTimeframe,
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
        <SourceBadge source={current.source} />
        <span>Generated {formatDateTime(current.generatedAt)}</span>
        {#if isLoading}<Icon name="refresh" size={12} class="animate-spin" />{/if}
      </p>
      {#if current.source?.stale}
        <p class="mt-1 text-xs text-orange-700 dark:text-orange-300">
          Every provider failed; these candles are the last good copy from {formatDateTime(current.source.asOf)}.
        </p>
      {:else if current.source?.fallback?.length}
        <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">Fallback: {current.source.fallback.join('; ')}</p>
      {/if}
    </div>
  </div>

  <RuleBasedNote class="mb-4" />

  <div class="grid gap-4 lg:grid-cols-3">
    <!-- WM-INV-12 -->
    <TechnicalsCard {technicals} class="lg:col-span-2" />

    <LevelsPanel {technicals} />
  </div>

  <ConfluenceCard confluence={current.confluence} timeframe={current.timeframe} />

  <SeasonalityCard buckets={current.seasonality} />

  <RiskCard risk={current.risk} />

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
