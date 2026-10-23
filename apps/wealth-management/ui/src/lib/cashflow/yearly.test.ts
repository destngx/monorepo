import { describe, expect, it } from 'vitest';
import type { VerdictStatus } from '$lib/api/types';
import {
  columnTotals,
  comparisonSpan,
  expenseSplitMessage,
  parseYear,
  projectionBasis,
  savingsRateDomain,
  verdictMessage,
} from './yearly';

describe('parseYear', () => {
  it('accepts a four-digit year and ignores anything else', () => {
    expect(parseYear('2025')).toBe(2025);
    expect(parseYear(null)).toBeUndefined();
    expect(parseYear('')).toBeUndefined();
    expect(parseYear('25')).toBeUndefined();
    expect(parseYear('2025-10')).toBeUndefined();
  });
});

describe('comparisonSpan', () => {
  it('names the whole previous year once the year is over', () => {
    expect(comparisonSpan(2025, 12)).toBe('2024');
  });

  it('names the same months of the previous year while the year runs', () => {
    expect(comparisonSpan(2026, 10)).toBe('Jan-Oct 2025');
    expect(comparisonSpan(2026, 1)).toBe('Jan 2025');
  });

  it('has nothing to compare for a year that has not started', () => {
    expect(comparisonSpan(2027, 0)).toBeNull();
  });
});

describe('columnTotals', () => {
  it('sums each month across the categories', () => {
    expect(
      columnTotals([
        { category: 'Food', values: [1, 2, 0], total: 3 },
        { category: 'Rent', values: [4, 0, 5], total: 9 },
      ]),
    ).toEqual([5, 2, 5]);
    expect(columnTotals([])).toEqual(Array(12).fill(0));
  });
});

describe('verdictMessage', () => {
  const target = { watch: 10, healthy: 20, target: 30 };
  const verdict = (status: VerdictStatus, savingsRate: number, deficitMonths = 0) => ({
    status,
    savingsRate,
    deficitMonths,
  });

  it('explains each status in plain words with the bands', () => {
    expect(verdictMessage(2025, verdict('onTarget', 34.2), target, 12)).toMatchObject({
      tone: 'good',
      title: 'On target',
      text: 'You kept 34.2% of your income, at or above the 30% target.',
    });
    expect(verdictMessage(2025, verdict('healthy', 24), target, 12).text).toBe(
      'You kept 24.0% of your income. 20-30% is healthy; 30% is the target.',
    );
    expect(verdictMessage(2025, verdict('watch', 12), target, 12)).toMatchObject({
      tone: 'warning',
      title: 'Worth watching',
    });
    expect(verdictMessage(2025, verdict('atRisk', 4), target, 12)).toMatchObject({ tone: 'serious', title: 'At risk' });
    expect(verdictMessage(2025, verdict('noData', 0), target, 12)).toMatchObject({
      tone: 'neutral',
      text: 'No income recorded for 2025 yet.',
    });
  });

  it('calls out spending more than earned', () => {
    expect(verdictMessage(2025, verdict('atRisk', -25), target, 12)).toMatchObject({
      tone: 'critical',
      title: 'Spending more than you earn',
      text: 'You spent 25.0% more than you earned.',
    });
  });

  it('counts the months that ran a deficit', () => {
    expect(verdictMessage(2026, verdict('healthy', 22, 3), target, 10).detail).toBe(
      '3 of 10 months spent more than they earned.',
    );
    expect(verdictMessage(2026, verdict('healthy', 22, 1), target, 10).detail).toBe(
      '1 of 10 months spent more than it earned.',
    );
    expect(verdictMessage(2026, verdict('healthy', 22, 0), target, 10).detail).toBe(
      'Every month earned more than it spent.',
    );
  });
});

describe('projectionBasis', () => {
  it('says what the year-end estimate is based on', () => {
    expect(projectionBasis(9)).toBe('Months so far, the rest at your Jan-Sep average.');
    expect(projectionBasis(1)).toBe('Months so far, the rest at your Jan average.');
    expect(projectionBasis(0)).toBe('This month so far, repeated for the year. Rough until a month has closed.');
  });
});

describe('savingsRateDomain', () => {
  it('always shows zero and the target band', () => {
    expect(savingsRateDomain([5, 12], 30)).toEqual([0, 40]);
  });

  it('follows the data but clamps runaway months', () => {
    expect(savingsRateDomain([-35, 72], 30)).toEqual([-40, 80]);
    expect(savingsRateDomain([-900, 100], 30)).toEqual([-100, 100]);
  });
});

describe('expenseSplitMessage', () => {
  const months = (total: number) => ({ category: '', values: [], total });
  // Shares are of income in percent: needs (fixed + essential + untagged), wants, saved.
  const split = (
    fixed: number,
    essential: number,
    flexible: number,
    untagged: number,
    shares: [number, number, number] | null,
    untaggedCategories: string[] = [],
  ) => ({
    fixed: months(fixed),
    essential: months(essential),
    flexible: months(flexible),
    untagged: months(untagged),
    needsShareOfIncome: shares?.[0] ?? null,
    needsShareComfortable: 50,
    wantsShareOfIncome: shares?.[1] ?? null,
    savingsShareOfIncome: shares?.[2] ?? null,
    savingsShareTarget: 30,
    untaggedCategories,
    room: [],
    roomPerMonth: 0,
    roomPerYear: 0,
  });

  it('asks for tags until any category is tagged', () => {
    expect(expenseSplitMessage(split(0, 0, 0, 100, [100, 0, 0], ['Food']))).toMatchObject({
      tone: 'neutral',
      title: 'Tag your categories',
    });
  });

  it('leaves room only when needs stay within 50% and saving meets 30%', () => {
    expect(expenseSplitMessage(split(30, 10, 15, 0, [40, 15, 45]))).toMatchObject({
      tone: 'good',
      title: 'Needs leave room',
      text: 'Needs take 40.0% of your income, wants 15.0%, and you save 45.0%. Needs are within 50% and saving meets the 30% target. Keep saving first: wants can grow only to 30.0% of income.',
      detail: null,
    });
  });

  it('says wants take the savings when needs are light but saving is short', () => {
    expect(expenseSplitMessage(split(30, 4, 50, 0, [34, 50, 16]))).toMatchObject({
      tone: 'warning',
      title: 'Wants take the savings',
      text: 'Needs take 34.0% of your income, wants 50.0%, and you save 16.0%. Needs are within 50%, but saving is under the 30% target. Bring wants down to 36.0% of income to reach it.',
    });
    expect(expenseSplitMessage(split(30, 10, 70, 0, [40, 70, -10])).tone).toBe('critical');
  });

  it('says needs are heavy above half of income', () => {
    expect(expenseSplitMessage(split(40, 20, 10, 0, [60, 10, 30]))).toMatchObject({
      tone: 'warning',
      title: 'Needs are heavy',
    });
    expect(expenseSplitMessage(split(60, 0, 10, 0, null)).text).toBe(
      'No income recorded, so there is no share to judge yet.',
    );
  });

  it('says untagged spending counts as needs until tagged', () => {
    expect(expenseSplitMessage(split(40, 0, 35, 25, [65, 35, 0], ['Gifts', 'Repairs'])).detail).toBe(
      '2 untagged categories (25.0% of spending) count as needs until tagged.',
    );
    expect(expenseSplitMessage(split(40, 0, 35, 25, [65, 35, 0], ['Gifts'])).detail).toBe(
      '1 untagged category (25.0% of spending) counts as needs until tagged.',
    );
  });
});
