import { arrayAt, dataOf, dateOf, num, objectAt } from './parse';

// Time series and ranges

export interface Point {
  date: string;
  value: number;
}

export type Range = '1M' | '3M' | '6M' | 'YTD' | '1Y' | '3Y' | '5Y' | 'ALL';
export const RANGES: Range[] = ['1M', '3M', '6M', 'YTD', '1Y', '3Y', '5Y', 'ALL'];
export type RangeHandler = (range: Range) => void;
export type ValueFormat = (value: number) => string;

const rangeMonths: Partial<Record<Range, number>> = { '1M': 1, '3M': 3, '6M': 6, '1Y': 12, '3Y': 36, '5Y': 60 };

/** First calendar day ("YYYY-MM-DD") a range covers up to `today`; null for ALL. */
export function rangeStart(range: Range, today: string): string | null {
  const [year, month, day] = today.split('-').map(Number);
  if (range === 'ALL') return null;
  if (range === 'YTD') return `${year}-01-01`;
  const months = rangeMonths[range] ?? 12;
  // Clamp the day so 31 March minus one month is 28/29 February, not 3 March.
  const target = new Date(Date.UTC(year, month - 1 - months, 1));
  const lastDay = new Date(Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0)).getUTCDate();
  target.setUTCDate(Math.min(day, lastDay));
  return target.toISOString().slice(0, 10);
}

/** Earliest date asked for when a history range is ALL; fmarket gold starts in 2000. */
export const HISTORY_ORIGIN = '2000-01-01';

/** fromDate/toDate params (yyyyMMdd) for the gold and USD history actions. */
export function historyParams(range: Range, today: string): { fromDate: string; toDate: string } {
  const compact = (date: string) => date.replaceAll('-', '');
  return { fromDate: compact(rangeStart(range, today) ?? HISTORY_ORIGIN), toDate: compact(today) };
}

/** Points on or after the range start; points must be oldest first. */
export function sliceRange<T extends { date: string }>(points: T[], range: Range, today: string): T[] {
  const start = rangeStart(range, today);
  return start ? points.filter((point) => point.date >= start) : points;
}

/** Sorts oldest first and keeps the last row of each day. */
export function ascendingByDate<T extends { date: string }>(rows: T[]): T[] {
  const byDate = new Map<string, T>();
  for (const row of [...rows].sort((a, b) => a.date.localeCompare(b.date))) byDate.set(row.date, row);
  return [...byDate.values()];
}

/** getNavHistory rows ({nav, navDate}) as NAV points, oldest first. */
export function parseNavHistory(payload: unknown): Point[] {
  const points = arrayAt(dataOf(payload)).flatMap((raw) => {
    const row = objectAt(raw);
    const date = dateOf(row.navDate) ?? dateOf(row.createdAt);
    const value = num(row.nav);
    return date && value !== null && value > 0 ? [{ date, value }] : [];
  });
  return ascendingByDate(points);
}

/** Percent change from the first to the last point. */
export function periodReturn(points: Point[]): number | null {
  if (points.length < 2 || points[0].value === 0) return null;
  return (points[points.length - 1].value / points[0].value - 1) * 100;
}

/** Keeps about `max` evenly spaced points, always including the first and last. */
export function downsample<T>(points: T[], max = 400): T[] {
  if (points.length <= max || max < 2) return points;
  const step = (points.length - 1) / (max - 1);
  return Array.from({ length: max }, (_, index) => points[Math.round(index * step)]);
}
