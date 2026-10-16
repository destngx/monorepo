<script lang="ts">
  import type { ThinkTankContext } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Money from '$lib/components/Money.svelte';
  import { biasTone, formatTimestamp, sourceHost } from '../news';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { rateDetail } from '$lib/rates';
  import { mask } from '$lib/state/mask.svelte';

  // What the models saw, disclosed as the engine sent it (WM-INV-18).
  let { context }: { context: ThinkTankContext } = $props();

  const nf = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 2 });
  const totals = $derived(context.valuation?.totals);
  const json = $derived(JSON.stringify(context, null, 2));
</script>

<div class="space-y-5 text-sm">
  <div class="grid gap-4 md:grid-cols-3">
    <div>
      <h3 class="label">Portfolio totals</h3>
      {#if totals}
        <dl class="space-y-0.5">
          <div class="flex justify-between gap-2">
            <dt class="muted">Total</dt>
            <dd><Money value={totals.total} /></dd>
          </div>
          <div class="flex justify-between gap-2">
            <dt class="muted">VND assets</dt>
            <dd><Money value={totals.vnd} /></dd>
          </div>
          <div class="flex justify-between gap-2">
            <dt class="muted">Crypto</dt>
            <dd><Money value={totals.crypto} /></dd>
          </div>
          {#if totals.unvalued}<p class="text-xs text-amber-700 dark:text-amber-300">
              {totals.unvalued} items not valued
            </p>{/if}
        </dl>
      {:else}
        <p class="text-amber-700 dark:text-amber-300">{context.valuationError || 'Not available'}</p>
      {/if}
    </div>
    <div>
      <h3 class="label">USDT rate and budget</h3>
      <dl class="space-y-0.5">
        <div class="flex justify-between gap-2">
          <dt class="muted">USDT/VND</dt>
          <dd class="tabular">
            {#if context.usdtRate}{nf.format(context.usdtRate.rate)}
              <SourceBadge
                source={context.usdtRate.source ?? { provider: context.usdtRate.provider }}
                detail={rateDetail(context.usdtRate)}
              />{:else}<span class="muted">N/A</span>{/if}
          </dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="muted">Planned income</dt>
          <dd><Money value={context.budget.income} /></dd>
        </div>
        <div class="flex justify-between gap-2">
          <dt class="muted">Planned expenses</dt>
          <dd><Money value={context.budget.expenses} /></dd>
        </div>
      </dl>
      <p class="muted mt-1 text-xs">
        {context.accounts.length} accounts, {context.moneyLent.open.length} open money-lent balances, {context
          .recentTransactions.length} recent transactions
      </p>
    </div>
    <div>
      <h3 class="label">Market pulse</h3>
      {#if context.pulse}
        <p class="font-medium">
          {context.pulse.scenario.name} <span class="muted text-xs">{context.pulse.scenario.regime}</span>
        </p>
        <p class="mt-0.5 text-xs">{context.pulse.scenario.summary}</p>
        <p class="mt-1 flex flex-wrap items-center gap-1.5 text-xs">
          <Badge tone="violet">{context.pulse.capitalFlow.signal}</Badge>
          {context.pulse.capitalFlow.summary}
        </p>
      {:else}
        <p class="text-amber-700 dark:text-amber-300">{context.pulseError || 'Not available'}</p>
      {/if}
      {#if context.pulse && context.pulseError}
        <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{context.pulseError}</p>
      {/if}
    </div>
  </div>

  <div>
    <h3 class="label">Web searches</h3>
    {#if context.searches?.length}
      <ul class="space-y-3">
        {#each context.searches as search, i (i)}
          <li>
            <p class="font-medium">
              “{search.query}”
              <span class="muted text-xs font-normal"
                >{search.provider}{search.fallback ? ` · fallback: ${search.fallback}` : ''}</span
              >
            </p>
            {#if search.results.length}
              <ul class="mt-1 list-disc space-y-0.5 pl-5 text-xs">
                {#each search.results as result, j (j)}
                  <li>
                    <a
                      class="text-indigo-600 hover:underline dark:text-indigo-400"
                      href={result.url}
                      target="_blank"
                      rel="noopener noreferrer">{result.title || result.url}</a
                    >
                    <span class="muted">{sourceHost(result)}</span>
                  </li>
                {/each}
              </ul>
            {:else}
              <p class="muted text-xs">No results.</p>
            {/if}
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted">No web search results were available.</p>
    {/if}
    {#if context.searchErrors?.length}
      <ul class="mt-2 space-y-0.5 text-xs text-amber-700 dark:text-amber-300">
        {#each context.searchErrors as error, i (i)}<li>{error}</li>{/each}
      </ul>
    {/if}
  </div>

  {#if context.news?.length}
    <div>
      <h3 class="label">News reports included</h3>
      <ul class="flex flex-wrap gap-2 text-xs">
        {#each context.news as report, i (i)}
          <li class="flex items-center gap-1.5">
            {report.topic}
            <Badge tone={biasTone[report.bias]}>{report.bias}</Badge>
            <span class="muted">{formatTimestamp(report.generatedAt)}</span>
          </li>
        {/each}
      </ul>
    </div>
  {/if}

  <div>
    <h3 class="label">Search context sent to the models</h3>
    <pre
      class="max-h-80 overflow-auto rounded-lg bg-slate-900/5 p-3 text-xs whitespace-pre-wrap dark:bg-white/5">{context.searchContext ||
        '(empty)'}</pre>
  </div>

  <details>
    <summary class="label cursor-pointer select-none">Full context JSON</summary>
    {#if mask.revealed}
      <pre class="mt-1 max-h-96 overflow-auto rounded-lg bg-slate-900/5 p-3 text-xs dark:bg-white/5">{json}</pre>
    {:else}
      <p class="muted mt-1 text-xs">
        Hidden while amounts are masked: it holds balances and transactions. Reveal amounts to view it.
      </p>
    {/if}
  </details>
</div>
