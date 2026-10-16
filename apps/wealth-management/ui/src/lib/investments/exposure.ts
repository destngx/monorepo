import type { DailyClose, DrawdownScenario, Exposure, HedgeTrend, Hedges, PriceMove } from '$lib/api/types';
import type { StatusTone } from '$lib/status';

// The engine computes exposure, the crypto view and the gold/USD trends
// (ROADMAP M1, M2, M4); this module only words and shapes them.

/** How much of the volatile money today's change is based on. */
export type Coverage = 'none' | 'partial' | 'full';

export function coverage(exposure: Exposure): Coverage {
  if (exposure.change === null) return 'none';
  // Within 1 VND counts as the whole volatile value (float noise).
  return exposure.covered + 1 >= exposure.volatileValue ? 'full' : 'partial';
}

/** "over about 24h" / "since the previous NAV" for a move. */
export function moveBasis(move: PriceMove): string {
  return move.basis === '24h' ? 'over about 24h' : 'since the previous NAV';
}

/**
 * How hard a crypto drawdown hits net worth, as a status with its own label (never colour alone):
 * under 10% of net worth is small, under 25% noticeable, more is large.
 */
export function drawdownStatus(scenario: DrawdownScenario): { tone: StatusTone; label: string } {
  if (scenario.netWorthChangePct === null) return { tone: 'neutral', label: 'No net worth to compare' };
  const hit = Math.abs(scenario.netWorthChangePct);
  if (hit < 10) return { tone: 'good', label: 'Small hit' };
  if (hit < 25) return { tone: 'warning', label: 'Noticeable hit' };
  return { tone: 'serious', label: 'Large hit' };
}

/** The periods the hedge chart can show, longest first. */
export const HEDGE_RANGES = ['1Y', '3M'] as const;
export type HedgeRange = (typeof HEDGE_RANGES)[number];

/** The change of a trend over a period, if the engine reported one. */
export function trendChange(trend: HedgeTrend | null, period: HedgeRange) {
  return trend?.changes.find((change) => change.period === period) ?? null;
}

/**
 * A trend rebased to 0% at the start of the range: each close as the percent change since the
 * period's reference close, so gold and USD share one percent axis.
 */
export function rebase(trend: HedgeTrend, period: HedgeRange): { date: string; value: number }[] {
  const change = trendChange(trend, period);
  if (!change || change.from <= 0) return [];
  return trend.points
    .filter((point: DailyClose) => point.date >= change.fromDate)
    .map((point) => ({ date: point.date, value: ((point.close - change.from) / change.from) * 100 }));
}

/** "gold rose 5.0% and the US dollar fell 1.0% over 12 months", or null without either trend. */
export function hedgeYearText(hedges: Hedges): string | null {
  const parts = [
    { name: 'gold', change: trendChange(hedges.gold, '1Y') },
    { name: 'the US dollar', change: trendChange(hedges.usd, '1Y') },
  ]
    .filter((part) => part.change !== null)
    .map(({ name, change }) => {
      const pct = change!.changePct;
      const amount = pct === 0 ? '' : ` ${Math.abs(pct).toFixed(1)}%`;
      return `${name} ${movedWord(pct)}${amount}`;
    });
  if (!parts.length) return null;
  const isPartial = [hedges.gold, hedges.usd].some((trend) => trendChange(trend, '1Y')?.partial);
  return `${parts.join(' and ')} over ${isPartial ? 'the available history' : '12 months'}`;
}

/** "rose" / "fell" / "held steady" for a percent change. */
export function movedWord(pct: number): string {
  if (pct > 0) return 'rose';
  if (pct < 0) return 'fell';
  return 'held steady';
}
