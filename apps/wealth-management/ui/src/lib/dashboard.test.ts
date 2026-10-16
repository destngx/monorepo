import { describe, expect, it } from 'vitest';
import { trendDelta, vndToUsd } from './dashboard';

describe('trendDelta', () => {
  it('returns the change between the first and last point', () => {
    const result = trendDelta([
      { date: '2026-09-14', value: 100 },
      { date: '2026-09-30', value: 90 },
      { date: '2026-10-13', value: 125 },
    ]);
    expect(result).toEqual({ delta: 25, percent: 25 });
  });

  it('has no percent from a zero start and no delta without two points', () => {
    expect(
      trendDelta([
        { date: 'a', value: 0 },
        { date: 'b', value: 10 },
      ]),
    ).toEqual({ delta: 10, percent: null });
    expect(trendDelta([{ date: 'a', value: 5 }])).toBeNull();
  });

  it('measures the percent against the magnitude of a negative start', () => {
    expect(
      trendDelta([
        { date: 'a', value: -100 },
        { date: 'b', value: -50 },
      ])?.percent,
    ).toBe(50);
  });
});

describe('vndToUsd', () => {
  it('divides by the VND-per-USD rate', () => {
    expect(vndToUsd(2_600_000, 26_000)).toBe(100);
  });

  it('is null for a missing or invalid rate', () => {
    expect(vndToUsd(1, null)).toBeNull();
    expect(vndToUsd(1, 0)).toBeNull();
    expect(vndToUsd(1, Number.NaN)).toBeNull();
  });
});
