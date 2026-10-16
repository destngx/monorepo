import type { CashFlow, DashboardSummary, ExpensePace } from './api/types';
import { formatPercent } from './format';

/**
 * Hint for the income and expense tiles (WM-DASH-2): month to date against
 * the same days of last month, or all of it when last month was shorter.
 */
export function sameDaysHint(summary: Pick<DashboardSummary, 'asOf' | 'comparedDays'>): string {
  const day = Number(summary.asOf.slice(8, 10));
  return day > summary.comparedDays ? ' vs all of last month' : ' vs same days last month';
}

/**
 * The savings rate tile is month to date. Before any income arrives the rate
 * means nothing, so it has none; the same days of last month give context.
 */
export function savingsRateTile(thisMonth: CashFlow, lastMonthToDate: CashFlow): { rate: number | null; hint: string } {
  if (thisMonth.income <= 0) return { rate: null, hint: 'month to date · no income yet' };
  const previous =
    lastMonthToDate.income > 0 ? ` · ${formatPercent(lastMonthToDate.savingsRate)} same days last month` : '';
  return { rate: thisMonth.savingsRate, hint: `month to date${previous}` };
}

export type ExpensePaceView =
  | { kind: 'pending'; fromDay: number; spent: number }
  | {
      kind: 'pace';
      projected: number;
      monthEnd: string;
      lastMonth: string;
      lastMonthExpense: number;
      changePercent: number | null;
      fixed: number;
    };

/** What the dashboard says about the month-end expense pace (WM-DASH-9). */
export function expensePaceView(pace: ExpensePace | null | undefined): ExpensePaceView | null {
  if (!pace) return null;
  if (pace.projected === null) return { kind: 'pending', fromDay: pace.minDays, spent: pace.spent };
  return {
    kind: 'pace',
    projected: pace.projected,
    monthEnd: pace.monthEnd,
    lastMonth: pace.lastMonth,
    lastMonthExpense: pace.lastMonthExpense,
    changePercent: pace.changePercent,
    fixed: pace.fixed,
  };
}
