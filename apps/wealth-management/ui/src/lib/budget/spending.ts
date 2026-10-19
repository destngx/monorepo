import type { SpendingDay, SpendingRecommendation, SpendingWindows } from '$lib/api/types';
import type { StatusTone } from '$lib/status';

/** The windows the card switches between (WM-BUD-12). */
export type SpendingRange = '30d' | '90d';

export const SPENDING_RANGES: { value: SpendingRange; label: string; days: number }[] = [
  { value: '30d', label: 'Last 30 days', days: 30 },
  { value: '90d', label: 'Last 3 months', days: 90 },
];

/** The window for a range: the engine sends the last 30 days, then the last 3 months. */
export function spendingWindow(spending: SpendingWindows, range: SpendingRange) {
  return spending.windows[range === '30d' ? 0 : 1];
}

/** The days of a range, oldest first, and an axis label every `every` days (day of month). */
export function spendingDays(daily: SpendingDay[], range: SpendingRange): { days: SpendingDay[]; labels: string[] } {
  const count = range === '30d' ? 30 : 90;
  const every = range === '30d' ? 5 : 15;
  const days = daily.slice(-count);
  return {
    days,
    labels: days.map((day, index) => ((days.length - 1 - index) % every === 0 ? day.date.slice(8, 10) : '')),
  };
}

const TONE: Record<SpendingRecommendation['tone'], StatusTone> = { warning: 'warning', info: 'info', good: 'good' };
const TONE_LABEL: Record<SpendingRecommendation['tone'], string> = { warning: 'Act', info: 'Look', good: 'Good' };

export function recommendationStatus(recommendation: SpendingRecommendation): { tone: StatusTone; label: string } {
  return { tone: TONE[recommendation.tone], label: TONE_LABEL[recommendation.tone] };
}

/** A stable key for a recommendation in a list. */
export function recommendationKey(recommendation: SpendingRecommendation): string {
  return [recommendation.kind, recommendation.category ?? '', recommendation.tag ?? ''].join('|');
}
