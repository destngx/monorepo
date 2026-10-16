import type { TrendPoint } from './api/types';

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
