import { describe, expect, it } from 'vitest';
import type { Transaction } from '$lib/api/types';
import { budgetDate, categoryTransactions, dailySpend, detailEnd } from './detail';

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
});
