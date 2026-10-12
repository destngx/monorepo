import type { Transaction } from './transactions';

export interface CashbackRule {
  name: string;
  tag: string;
  rate: number;
  capMonthly: number;
  fallback?: boolean;
}

export interface CardCycle {
  statementDay: number;
  dueDay: number;
  cycleStart: string;
  cycleEnd: string;
  cycleDays: number;
  cycleElapsedDays: number;
  cycleProgress: number;
  nextDueDate: string;
  daysUntilDue: number;
  dueUrgent: boolean;
  minimumPayment: number;
  outstandingAmount: number;
}

export interface CardHistoryMonth {
  month: string;
  spent: number;
  potentialCashback: number;
  actualRefund: number;
  efficiency: number;
  missed: number;
}

export type UtilizationBand = 'good' | 'fair' | 'high';

export interface CardReport {
  name: string;
  bank: string;
  keyword: string;
  accountName: string;
  expiry: string;
  limit: number;
  usage: number;
  utilization: number;
  utilizationBand: UtilizationBand;
  remainingLimit: number;
  spentThisCycle: number;
  estimatedCashback: number;
  refundYtd: number;
  annualFee: number;
  netEarnYtd: number;
  lifetimeFees: number;
  /** Card-level monthly cashback cap; 0 means none (WM-CARD-2). */
  monthlyCap: number;
  transactionCount: number;
  cycle: CardCycle;
  rules: CashbackRule[];
  history: CardHistoryMonth[];
  cycleTransactions: Transaction[];
}

export interface CardAccountGroup {
  name: string;
  shared: boolean;
  limit: number;
  usage: number;
  utilization: number;
  remainingLimit: number;
  estimatedCashback: number;
  refundYtd: number;
  annualFees: number;
  netEarnYtd: number;
  cards: CardReport[];
}

export interface CardBankReport {
  name: string;
  accounts: CardAccountGroup[];
}
