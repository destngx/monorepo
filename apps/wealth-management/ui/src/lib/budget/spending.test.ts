import { describe, expect, it } from 'vitest';
import type { SpendingDay } from '$lib/api/types';
import { recommendationKey, recommendationStatus, spendingDays } from './spending';

const daily: SpendingDay[] = Array.from({ length: 90 }, (_, index) => ({
  date: new Date(Date.UTC(2026, 6, 22 + index)).toISOString().slice(0, 10),
  fixed: 0,
  dayToDay: index,
}));

describe('spending windows', () => {
  it('Given 90 days When the last 30 are shown Then they end today with a label every 5 days counted back from today', () => {
    const { days, labels } = spendingDays(daily, '30d');
    expect(days).toHaveLength(30);
    expect(days[29].date).toBe('2026-10-19');
    expect(labels[29]).toBe('19');
    expect(labels[24]).toBe('14');
    expect(labels.filter(Boolean)).toHaveLength(6);
  });

  it('Given 90 days When the 3 months are shown Then all show with a label every 15 days', () => {
    const { days, labels } = spendingDays(daily, '90d');
    expect(days).toHaveLength(90);
    expect(labels.filter(Boolean)).toHaveLength(6);
  });

  it('Given a recommendation When it is shown Then its tone has a word, not colour alone, and a stable key', () => {
    const advice = {
      kind: 'split_rising',
      tone: 'warning',
      category: 'Food',
      tag: 'eating-out',
      current: 1,
      usual: 1,
      amount: 1,
      changePercent: 1,
    } as const;
    expect(recommendationStatus(advice)).toEqual({ tone: 'warning', label: 'Act' });
    expect(recommendationStatus({ ...advice, tone: 'good' }).label).toBe('Good');
    expect(recommendationKey(advice)).toBe('split_rising|Food|eating-out');
  });
});
