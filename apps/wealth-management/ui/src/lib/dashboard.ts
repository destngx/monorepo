import type { BudgetStatus, TrendPoint } from './api/types';

/** Change between the first and last point of a trend (WM-DASH-7: 30-day delta). */
export function trendDelta(points: TrendPoint[]): { delta: number; percent: number | null } | null {
  if (points.length < 2) return null;
  const first = points[0].value;
  const last = points[points.length - 1].value;
  const delta = last - first;
  return { delta, percent: first === 0 ? null : (delta / Math.abs(first)) * 100 };
}

/** VND amount in USD at `vndPerUsd`; null when the rate is unusable. */
export function vndToUsd(vnd: number, vndPerUsd: number | null | undefined): number | null {
  if (!vndPerUsd || !Number.isFinite(vndPerUsd) || vndPerUsd <= 0) return null;
  return vnd / vndPerUsd;
}

// WM-RULE-6 bands, decision D4: >= 90% urgent, >= 70% warning, else healthy.
export const BUDGET_WARNING_PERCENT = 70;
export const BUDGET_URGENT_PERCENT = 90;

export const BUDGET_STATUS_BAR: Record<BudgetStatus, string> = {
  healthy: 'bg-emerald-500',
  warning: 'bg-amber-400',
  urgent: 'bg-rose-500',
};

export function budgetStatusFor(percentUsed: number): BudgetStatus {
  if (percentUsed >= BUDGET_URGENT_PERCENT) return 'urgent';
  if (percentUsed >= BUDGET_WARNING_PERCENT) return 'warning';
  return 'healthy';
}

/** Bar colour for a budget line; the engine's status wins over the local D4 fallback. */
export function budgetBarClass(line: { status?: BudgetStatus | null; percentUsed: number }): string {
  const status = line.status && line.status in BUDGET_STATUS_BAR ? line.status : budgetStatusFor(line.percentUsed);
  return BUDGET_STATUS_BAR[status];
}
