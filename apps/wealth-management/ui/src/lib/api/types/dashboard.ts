import type { Account } from './accounts';

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
