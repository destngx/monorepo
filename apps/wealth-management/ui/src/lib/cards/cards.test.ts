import { describe, expect, it } from 'vitest';
import type { CardHistoryMonth, CashbackRule } from '$lib/api/types';
import { cardMonthlyCap, compareCards, cumulative, historyYears } from './cards';

const rule = (capMonthly: number): CashbackRule => ({ name: 'r', tag: 't', rate: 1, capMonthly });
const month = (m: string, actualRefund: number): CardHistoryMonth => ({
  month: m,
  spent: actualRefund * 100,
  potentialCashback: actualRefund * 2,
  actualRefund,
  efficiency: 50,
  missed: actualRefund,
});

describe('cardMonthlyCap', () => {
  it('prefers the engine card-level cap', () => {
    expect(cardMonthlyCap({ monthlyCap: 300_000, rules: [rule(0)] })).toBe(300_000);
  });

  it('sums rule caps only when every rule is capped', () => {
    expect(cardMonthlyCap({ rules: [rule(100), rule(200)] })).toBe(300);
    expect(cardMonthlyCap({ rules: [rule(100), rule(0)] })).toBeNull();
    expect(cardMonthlyCap({ rules: [] })).toBeNull();
  });
});

describe('cumulative', () => {
  it('returns the running total', () => {
    expect(cumulative([1, 2, 3])).toEqual([1, 3, 6]);
    expect(cumulative([])).toEqual([]);
  });
});

describe('cross-card comparison', () => {
  const cards = [
    { name: 'Visa', history: [month('2026-02', 20), month('2026-01', 10), month('2025-12', 5)] },
    { name: 'Master', history: [month('2026-03', 7)] },
  ];

  it('lists years newest first', () => {
    expect(historyYears(cards)).toEqual(['2026', '2025']);
  });

  it('aligns every card on the union of months with gaps as null', () => {
    expect(compareCards(cards, '2026', 'actualRefund')).toEqual({
      months: ['2026-01', '2026-02', '2026-03'],
      series: [
        { name: 'Visa', values: [10, 20, null] },
        { name: 'Master', values: [null, null, 7] },
      ],
    });
  });
});
