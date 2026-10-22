<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import { defer, type Streamed } from '$lib/api/load';
  import type { MarketAnalysis, Timeframe } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import LoadError from '$lib/components/LoadError.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonText from '$lib/components/skeleton/SkeletonText.svelte';
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
  // The symbol form and quick picks show at once; the analysis streams in
  // below (WM-UI-16). Another symbol or timeframe shows the skeleton, never
  // the old figures; a refresh keeps them with a Refreshing badge.
  const symbol = $derived(parseSymbol(page.url.searchParams.get('symbol')));
  const timeframe = $derived(parseTimeframe(page.url.searchParams.get('tf')));
  const scope = $derived(`${symbol}|${timeframe}`);

  // Bumped by Refresh and Retry, which skip the engine cache.
  let reloads = $state(0);
  let forceNext = false;
  const analysisData = $derived.by<Streamed<MarketAnalysis> | null>(() => {
    const [sym, tf] = [symbol, timeframe];
    void reloads;
    const force = forceNext;
    forceNext = false;
    return sym ? defer(() => api.marketAnalysis(sym, tf, { force })) : null;
  });
  const analysis = track(() => analysisData, { scope: () => scope });

  function reload(): Streamed<MarketAnalysis> | null {
    forceNext = true;
    reloads++;
    return analysisData;
  }

  // Busy until the analysis answers; a failure toasts and shows under the old figures.
  async function refresh() {
    const result = await reload();
    if (result && !result.ok) throw new Error(result.error);
  }

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
  const draftSymbol = $derived(parseSymbol(draft));
  // The Analyse button is busy while the symbol it asked for loads.
  const analysing = $derived(!!symbol && analysis.pending && draftSymbol === symbol);

  function submit(event: SubmitEvent) {
    event.preventDefault();
    if (draftSymbol && draftSymbol !== symbol) navigate(draftSymbol, timeframe);
    else if (draftSymbol) reload();
  }
</script>

<PageHeader
  title="Ticker Analysis"
  subtitle="Technicals, multi-timeframe confluence, seasonality and history-only risk"
>
  {#snippet actions()}
    <TimeframeSwitcher bind:value={() => timeframe, (tf) => navigate(symbol, tf)} />
    <Button
      icon="refresh"
      busy={analysis.refreshing}
      busyText="Refreshing…"
      disabled={!symbol || analysis.loading}
      errorContext="Could not refresh the analysis"
      onclick={refresh}>Refresh</Button
    >
  {/snippet}
</PageHeader>

<div class="panel mb-4 p-4">
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
        aria-invalid={draft.trim() !== '' && !draftSymbol}
        bind:value={() => draft, (value) => (draft = value.toUpperCase())}
      />
    </div>
    <Button type="submit" variant="primary" icon="search" busy={analysing} busyText="Analysing…" disabled={!draftSymbol}
      >Analyse</Button
    >
  </form>
  <div class="mt-3 flex flex-wrap gap-1.5" role="group" aria-label="Quick picks">
    {#each PULSE_SYMBOLS as pick (pick)}
      <button
        type="button"
        class="rounded-md px-2.5 py-0.5 text-xs font-medium transition active:scale-95 {pick === symbol
          ? 'bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900'
          : 'bg-stone-900/5 text-stone-700 hover:bg-stone-900/10 hover:text-stone-900 dark:bg-white/10 dark:text-stone-300 dark:hover:bg-white/15 dark:hover:text-white'}"
        aria-pressed={pick === symbol}
        onclick={() => navigate(pick, timeframe)}>{pick}</button
      >
    {/each}
  </div>
</div>

{#if !symbol}
  <div class="panel">
    <EmptyState title="Pick a ticker" detail="Enter a symbol or choose one of the market pulse instruments above." />
  </div>
{:else}
  <Deferred data={analysisData} {scope} label="the {symbol} analysis" retry={reload}>
    {#snippet children(current)}
      {@const technicals = current.technicals}
      <!-- Header -->
      <div class="panel mb-4 flex flex-wrap items-end justify-between gap-4 p-5">
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
          </p>
          {#if current.source?.stale}
            <p class="mt-1 text-xs text-orange-700 dark:text-orange-300">
              Every provider failed; these candles are the last good copy from {formatDateTime(current.source.asOf)}.
            </p>
          {:else if current.source?.fallback?.length}
            <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">
              Fallback: {current.source.fallback.join('; ')}
            </p>
          {/if}
        </div>
      </div>

      <RuleBasedNote class="mb-4" />

      <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
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
        load={(force, signal) => api.tickerInsight(current.symbol, current.timeframe, { force, signal })}
      />
    {/snippet}
    {#snippet error(failure, retry)}
      <div>
        <LoadError {failure} label="the {symbol} analysis on {TIMEFRAME_LABELS[timeframe]}" {retry} />
        {#if symbol === 'SJC' && (timeframe === '1h' || timeframe === '4h')}
          <p class="muted mt-2 text-sm">SJC gold is only published daily; try the 1D or 1W timeframe.</p>
        {/if}
      </div>
    {/snippet}
    {#snippet skeleton()}
      <div class="space-y-4">
        <section class="panel p-5" aria-label={symbol}>
          <h2 class="text-2xl font-semibold tracking-tight">{symbol}</h2>
          <SkeletonText lines={2} label="the {symbol} analysis" class="mt-3 max-w-sm" />
        </section>
        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <SkeletonCard title="Technicals" lines={4} chart={160} class="lg:col-span-2" />
          <SkeletonCard title="Levels" lines={6} />
        </div>
        <SkeletonCard title="Confluence" lines={3} />
      </div>
    {/snippet}
  </Deferred>
{/if}
