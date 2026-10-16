import type { Exposure, HedgeTrend, Hedges } from '$lib/api/types';
import { describe, expect, it } from 'vitest';
import { coverage, drawdownStatus, hedgeYearText, movedWord, rebase, trendChange } from './exposure';

const exposure = (overrides: Partial<Exposure> = {}): Exposure => ({
  total: 200_000_000,
  netWorth: 150_000_000,
  volatileValue: 130_000_000,
  volatileShare: 65,
  change: 1_000_000,
  changePct: 1,
  covered: 130_000_000,
  classes: [],
  crypto: {
    value: 0,
    holdings: 0,
    accounts: 0,
    share: 0,
    shareOfNetWorth: null,
    coins: [],
    change: null,
    scenarios: [],
  },
  funds: [],
  issues: [],
  generatedAt: '2026-10-17T00:00:00Z',
  ...overrides,
});

const trend = (overrides: Partial<HedgeTrend> = {}): HedgeTrend => ({
  asset: 'gold',
  label: 'SJC gold',
  symbol: 'SJC',
  unit: 'VND per tael (SJC sell price)',
  latest: 120,
  latestDate: '2026-10-16',
  changes: [
    { period: '3M', fromDate: '2026-07-16', from: 110, changePct: 9.09, partial: false, onCash: 9_090_000 },
    { period: '1Y', fromDate: '2025-10-16', from: 100, changePct: 20, partial: false, onCash: 20_000_000 },
  ],
  points: [
    { date: '2025-10-16', close: 100 },
    { date: '2026-07-16', close: 110 },
    { date: '2026-10-16', close: 120 },
  ],
  ...overrides,
});

const hedges = (overrides: Partial<Hedges> = {}): Hedges => ({
  cash: 100_000_000,
  cashShare: 50,
  vndShare: 75,
  gold: trend(),
  usd: trend({
    asset: 'usd',
    label: 'US dollar',
    changes: [{ period: '1Y', fromDate: '2025-10-16', from: 100, changePct: -1.5, partial: false, onCash: -1_500_000 }],
  }),
  issues: [],
  generatedAt: '2026-10-17T00:00:00Z',
  ...overrides,
});

describe('coverage', () => {
  it('says whether today’s change covers all, part or none of the volatile money', () => {
    expect(coverage(exposure())).toBe('full');
    expect(coverage(exposure({ covered: 100_000_000 }))).toBe('partial');
    expect(coverage(exposure({ change: null, covered: 0 }))).toBe('none');
  });
});

describe('drawdownStatus', () => {
  it('grades the hit to net worth with a label, never by colour alone', () => {
    const scenario = (pct: number | null) => ({ fallPct: 30, loss: 1, netWorthAfter: 1, netWorthChangePct: pct });
    expect(drawdownStatus(scenario(-5))).toEqual({ tone: 'good', label: 'Small hit' });
    expect(drawdownStatus(scenario(-15))).toEqual({ tone: 'warning', label: 'Noticeable hit' });
    expect(drawdownStatus(scenario(-30))).toEqual({ tone: 'serious', label: 'Large hit' });
    expect(drawdownStatus(scenario(null)).tone).toBe('neutral');
  });
});

describe('hedge trends', () => {
  it('rebases a trend to 0% at the period start so gold and USD share one axis', () => {
    expect(rebase(trend(), '1Y').map((p) => p.value)).toEqual([0, 10, 20]);
    expect(rebase(trend(), '3M')).toEqual([
      { date: '2026-07-16', value: 0 },
      { date: '2026-10-16', value: (10 / 110) * 100 },
    ]);
    expect(rebase(trend({ changes: [] }), '1Y')).toEqual([]);
    expect(trendChange(null, '1Y')).toBeNull();
  });

  it('words the 12-month change of each asset with its direction', () => {
    expect(hedgeYearText(hedges())).toBe('gold rose 20.0% and the US dollar fell 1.5% over 12 months');
    expect(hedgeYearText(hedges({ usd: null }))).toBe('gold rose 20.0% over 12 months');
    expect(hedgeYearText(hedges({ gold: null, usd: null }))).toBeNull();
    const partial = trend({ changes: [{ ...trend().changes[1], partial: true }] });
    expect(hedgeYearText(hedges({ gold: partial, usd: null }))).toBe('gold rose 20.0% over the available history');
    expect(movedWord(0)).toBe('held steady');
  });
});
