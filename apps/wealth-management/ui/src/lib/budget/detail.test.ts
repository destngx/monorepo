import { describe, expect, it } from 'vitest';
import type { Transaction } from '$lib/api/types';
import {
  budgetDate,
  categoryTransactions,
  dailySpend,
  dailySplitSpend,
  detailEnd,
  rowSplitTag,
  splitSeries,
} from './detail';

const tx = (date: string, category: string, payment: number | null, deposit: number | null = null) =>
  ({ id: date + category, date, category, payment, deposit, tags: [] }) as unknown as Transaction;

describe('budget category detail', () => {
  const rows = [
    tx('2026-10-05', 'Food', 100),
    tx('2026-10-05', 'food ', 50),
    tx('2026-10-06', 'Food', null, 30),
    tx('2026-09-30', 'Food', 70),
    tx('2026-10-05', 'Fun', 999),
  ];

  it('matches categories case-insensitively within the month', () => {
    expect(categoryTransactions(rows, 'FOOD', '2026-10')).toHaveLength(3);
  });

  it('groups by the effective date when the engine sends one', () => {
    const refund = { ...tx('2026-11-02', 'Food', null, 40), effectiveDate: '2026-10-20' } as Transaction;
    expect(budgetDate(refund)).toBe('2026-10-20');
    expect(budgetDate(tx('2026-11-02', 'Food', 1))).toBe('2026-11-02');
    expect(budgetDate({ date: '2026-11-02', effectiveDate: null })).toBe('2026-11-02');
    expect(categoryTransactions([...rows, refund], 'Food', '2026-10')).toContain(refund);
    expect(categoryTransactions([...rows, refund], 'Food', '2026-11')).toHaveLength(0);
  });

  it('nets refunds per day and never goes below zero', () => {
    const days = dailySpend(rows, 'Food', '2026-10-06');
    expect(days).toHaveLength(30);
    expect(days.at(-1)).toEqual({ date: '2026-10-06', value: 0 });
    expect(days.at(-2)).toEqual({ date: '2026-10-05', value: 150 });
    expect(days.at(-7)).toEqual({ date: '2026-09-30', value: 70 });
  });

  it('ends the window today in the current month, else on the last day', () => {
    expect(detailEnd('2026-10', '2026-10-12')).toBe('2026-10-12');
    expect(detailEnd('2026-02', '2026-10-12')).toBe('2026-02-28');
  });

  it('Given split tags When daily spend is divided Then each day is its tagged parts plus the rest', () => {
    const tagged = (date: string, payment: number | null, deposit: number | null, ...tags: string[]) => ({
      ...tx(date, 'Food', payment, deposit),
      id: date + payment + deposit,
      tags,
    });
    const ledger = [
      ...rows,
      tagged('2026-10-05', 300, null, 'card-cashback', 'Eating-Out'),
      tagged('2026-10-06', 40, null, 'takeaway', 'eating-out'),
      tagged('2026-10-06', null, 100, 'takeaway'),
    ];

    const days = dailySplitSpend(ledger, 'Food', '2026-10-06', ['eating-out', 'takeaway']);

    expect(days.at(-2)).toEqual({ date: '2026-10-05', rest: 150, tags: [300, 0] });
    // A row with both tags counts under the first; a refund never takes a part below zero.
    expect(days.at(-1)).toEqual({ date: '2026-10-06', rest: 0, tags: [40, 0] });
    expect(rowSplitTag(ledger[5], ['eating-out'])).toBe('eating-out');
    expect(rowSplitTag(rows[0], ['eating-out'])).toBeUndefined();
  });

  it('Given split tags When the daily chart is stacked Then the rest comes first and a third tag folds into other', () => {
    const days = [{ rest: 10, tags: [1, 2, 3] }];
    expect(splitSeries(days, [])).toEqual([{ name: 'Spend', part: 'rest', values: [10] }]);
    expect(splitSeries(days, ['eating-out', 'takeaway']).map((s) => s.name)).toEqual([
      'Rest of the category',
      '#eating-out',
      '#takeaway',
    ]);
    expect(splitSeries(days, ['a', 'b', 'c'])[2]).toEqual({ name: 'Other split tags', part: 'second', values: [5] });
  });
});
