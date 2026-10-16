<script lang="ts">
  import type { Snippet } from 'svelte';
  import type { Technicals } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import { actionTone, formatNumber, formatPrice, rsiTone, trendTone } from '../market';

  interface Props {
    technicals: Technicals;
    title?: string;
    subtitle?: string;
    /** Hides the signal reasons and trade levels for a summary view. */
    compact?: boolean;
    class?: string;
    actions?: Snippet;
  }

  let {
    technicals,
    title = 'Technicals',
    subtitle = 'Computed from candles',
    compact = false,
    class: className = '',
    actions,
  }: Props = $props();

  const indicators = $derived(technicals.indicators);
  const signal = $derived(technicals.signal);

  // EMA position relative to price, so the reader sees which averages are above.
  const emas = $derived(
    [
      { label: 'EMA 20', value: indicators.ema20 },
      { label: 'EMA 50', value: indicators.ema50 },
      { label: 'EMA 200', value: indicators.ema200 },
    ].map((ema) => ({ ...ema, above: ema.value === null ? null : technicals.price >= ema.value })),
  );
</script>

<Card {title} {subtitle} class={className} {actions}>
  <div class="flex flex-wrap items-center gap-2">
    <Badge tone={trendTone[technicals.trend]}>Trend: {technicals.trend}</Badge>
    <Badge tone="violet">Wyckoff: {technicals.phase}</Badge>
    <Badge tone={actionTone(signal.action)}>{signal.action}</Badge>
    <span class="muted text-xs">Confidence {signal.confidence}% · {technicals.candles} candles</span>
  </div>

  <dl class="mt-4 grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-3">
    <div>
      <dt class="muted text-xs">RSI 14</dt>
      <dd class="tabular font-medium">
        {#if indicators.rsi !== null}
          {formatNumber(indicators.rsi, 1)}
          {#if rsiTone(indicators.rsi) !== 'neutral'}
            <Badge tone={rsiTone(indicators.rsi)}>{indicators.rsi >= 70 ? 'Overbought' : 'Oversold'}</Badge>
          {/if}
        {:else}<span class="muted">-</span>{/if}
      </dd>
    </div>
    {#each emas as ema (ema.label)}
      <div>
        <dt class="muted text-xs">{ema.label}</dt>
        <dd class="tabular font-medium">
          {formatPrice(ema.value)}
          {#if ema.above !== null}
            <span class="muted text-xs font-normal">price {ema.above ? 'above' : 'below'}</span>
          {/if}
        </dd>
      </div>
    {/each}
    <div>
      <dt class="muted text-xs">ATR 14</dt>
      <dd class="tabular font-medium">{formatPrice(indicators.atr)}</dd>
    </div>
    <div class="col-span-2 sm:col-span-3">
      <dt class="muted text-xs">Bollinger (20, 2)</dt>
      <dd class="tabular font-medium">
        {#if indicators.bollinger}
          {formatPrice(indicators.bollinger.lower)} / {formatPrice(indicators.bollinger.middle)} / {formatPrice(
            indicators.bollinger.upper,
          )}
          <span class="muted text-xs font-normal">lower / middle / upper</span>
        {:else}<span class="muted">-</span>{/if}
      </dd>
    </div>
  </dl>

  {#if !compact}
    <div class="mt-4 rounded-xl border border-slate-900/10 p-3 dark:border-white/10">
      <p class="text-xs font-semibold tracking-wide text-slate-600 uppercase dark:text-slate-400">Signal</p>
      <dl class="mt-2 grid grid-cols-2 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
        <div>
          <dt class="muted text-xs">Entry</dt>
          <dd class="tabular font-medium">{formatPrice(signal.entry)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Stop loss</dt>
          <dd class="tabular font-medium text-rose-700 dark:text-rose-300">{formatPrice(signal.stopLoss)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Take profit</dt>
          <dd class="tabular font-medium text-emerald-700 dark:text-emerald-300">{formatPrice(signal.takeProfit)}</dd>
        </div>
        <div>
          <dt class="muted text-xs">Risk / reward</dt>
          <dd class="tabular font-medium">
            {signal.riskReward === undefined ? '-' : `1 : ${formatNumber(signal.riskReward)}`}
          </dd>
        </div>
      </dl>
      {#if signal.reasons.length}
        <ul class="mt-3 list-disc space-y-1 pl-5 text-sm">
          {#each signal.reasons as reason, i (i)}<li>{reason}</li>{/each}
        </ul>
      {/if}
    </div>
  {/if}
</Card>
