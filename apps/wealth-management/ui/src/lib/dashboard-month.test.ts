import { describe, expect, it } from 'vitest';
import type { CashFlow, ExpensePace } from './api/types';
import { expensePaceView, sameDaysHint, savingsRateTile } from './dashboard-month';

const flow = (income: number, expense: number, savingsRate = 0): CashFlow => ({
  month: '2026-10',
  income,
  expense,
  cashFlow: income - expense,
  savingsRate,
});

const pace = (overrides: Partial<ExpensePace> = {}): ExpensePace => ({
  daysElapsed: 10,
  daysInMonth: 31,
  monthEnd: '2026-10-31',
  minDays: 7,
  spent: 13_000_000,
  fixed: 10_000_000,
  variable: 3_000_000,
  projected: 19_300_000,
  lastMonth: '2026-09',
  lastMonthExpense: 19_000_000,
  changePercent: 1.58,
  ...overrides,
});

describe('sameDaysHint (WM-DASH-2)', () => {
  it('compares with the same days of last month', () => {
    expect(sameDaysHint({ asOf: '2026-10-12', comparedDays: 12 })).toBe(' vs same days last month');
  });

  it('says the whole month when last month was shorter than today', () => {
    expect(sameDaysHint({ asOf: '2026-03-31', comparedDays: 28 })).toBe(' vs all of last month');
  });
});

describe('savingsRateTile', () => {
  it('has no rate until income has arrived this month', () => {
    expect(savingsRateTile(flow(0, 2_000_000), flow(10_000_000, 1_000_000, 90))).toEqual({
      rate: null,
      hint: 'month to date · no income yet',
    });
  });

  it('labels the rate month to date with the same days of last month', () => {
    expect(savingsRateTile(flow(10_000_000, 4_000_000, 60), flow(10_000_000, 5_000_000, 50))).toEqual({
      rate: 60,
      hint: 'month to date · 50.0% same days last month',
    });
  });

  it('leaves out last month when it had no income by then', () => {
    expect(savingsRateTile(flow(10_000_000, 4_000_000, 60), flow(0, 5_000_000))).toEqual({
      rate: 60,
      hint: 'month to date',
    });
  });
});

describe('expensePaceView (WM-DASH-9)', () => {
  it('is hidden without a pace', () => {
    expect(expensePaceView(null)).toBeNull();
    expect(expensePaceView(undefined)).toBeNull();
  });

  it('waits for the minimum days before projecting', () => {
    expect(expensePaceView(pace({ daysElapsed: 3, projected: null, changePercent: null }))).toEqual({
      kind: 'pending',
      fromDay: 7,
      spent: 13_000_000,
    });
  });

  it('projects to the month end against last month', () => {
    expect(expensePaceView(pace())).toEqual({
      kind: 'pace',
      projected: 19_300_000,
      monthEnd: '2026-10-31',
      lastMonth: '2026-09',
      lastMonthExpense: 19_000_000,
      changePercent: 1.58,
      fixed: 10_000_000,
    });
  });
});
