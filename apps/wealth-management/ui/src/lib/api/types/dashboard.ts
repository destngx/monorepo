import type { Account } from './accounts';
import type { CategoryNature } from './budget';

export interface TrendPoint {
  date: string;
  value: number;
}

export interface BalanceSheet {
  assets: number;
  liabilities: number;
  accountsNet: number;
  loansRemaining: number;
  netWorth: number;
}

export interface CashFlow {
  month: string;
  income: number;
  expense: number;
  cashFlow: number;
  savingsRate: number;
}

export interface DashboardSummary {
  asOf: string;
  balanceSheet: BalanceSheet;
  netGrowthThisMonth: number;
  netWorthChangePercent: number | null;
  netWorthTrend: TrendPoint[];
  thisMonth: CashFlow;
  lastMonth: CashFlow;
  incomeChangePercent: number | null;
  expenseChangePercent: number | null;
  cashFlowHistory: CashFlow[];
  topAccounts: Account[];
}

/** One category's amount per month (January first) and its year total. */
export interface CategoryMonths {
  category: string;
  values: number[];
  total: number;
}

/** Income and expense of a calendar year (WM-DASH-8). */
export interface YearlyCashFlow {
  year: number;
  /** Years with data, newest first, plus the current and requested year. */
  years: number[];
  /** 12 for a past year, the current month for this year, 0 for a future one. */
  monthsElapsed: number;
  months: CashFlow[];
  total: CashFlow;
  monthlyAverage: CashFlow;
  /** The same elapsed months of the previous year. */
  previousYear: CashFlow;
  incomeChangePercent: number | null;
  expenseChangePercent: number | null;
  income: CategoryMonths[];
  expense: CategoryMonths[];
  verdict: YearVerdict;
  /** Year end at the current pace; only for the current year. */
  projection: YearProjection | null;
  /** Elapsed months whose expense stands out, oldest first. */
  spikes: ExpenseSpike[];
  savingsTarget: SavingsTarget;
  expenseSplit: ExpenseSplit;
}

/** Expense by the Categories nature tag (WM-DASH-8): needs are fixed plus essential, wants are flexible. */
export interface ExpenseSplit {
  fixed: CategoryMonths;
  essential: CategoryMonths;
  flexible: CategoryMonths;
  untagged: CategoryMonths;
  /** Fixed plus essential expense over income in percent; null without income. */
  needsShareOfIncome: number | null;
  needsShareComfortable: number;
  /** Expense categories with spending but no tag, largest first. */
  untaggedCategories: string[];
  /** Essential and flexible categories that could cost less, most room first; rows do not add up to the total. */
  room: CategoryRoom[];
  /** Whole months of essential + flexible spending, typical against lean. */
  roomPerMonth: number;
  roomPerYear: number;
}

/** Typical (median) month against the lean (25th percentile) one: a level already lived on. */
export interface CategoryRoom {
  category: string;
  nature: CategoryNature;
  typical: number;
  lean: number;
  perMonth: number;
  perYear: number;
}

export type VerdictStatus = 'onTarget' | 'healthy' | 'watch' | 'atRisk' | 'noData';

export interface YearVerdict {
  status: VerdictStatus;
  savingsRate: number;
  /** Elapsed months that spent more than they earned. */
  deficitMonths: number;
}

/** Savings rate bands in percent: watch from `watch`, healthy from `healthy`, on target from `target`. */
export interface SavingsTarget {
  watch: number;
  healthy: number;
  target: number;
}

export interface YearProjection extends CashFlow {
  /** Completed months the pace is averaged over; 0 in January (the month so far). */
  basisMonths: number;
}

export interface ExpenseSpike {
  month: string;
  expense: number;
  /** Median expense of the other months. */
  typical: number;
  ratio: number;
  /** The category that grew the most that month. */
  category: string;
  categoryAmount: number;
  categoryTypical: number;
}
