<script lang="ts">
  import type { RiskStats } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import { formatDate, formatPercent, formatSignedPercent } from '$lib/format';
  import { changeClass, formatPrice } from '../market';

  interface Props {
    risk: RiskStats | null;
  }

  let { risk }: Props = $props();
</script>

<Card title="Risk" subtitle="From daily history" class="mt-4">
  {#if risk}
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
          Bootstrap resample of past daily returns over {sim.horizonDays} trading days ({sim.paths.toLocaleString('en')} paths)
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
