export interface CardHistory {
  month: string;
  spent: number;
  cashback: number;
  actualRefund: number;
  efficiency: number;
}

export interface CardStat {
  name: string;
  bank: string;
  tagKeyword: string;
  defaultLimit: number;
  expiry: string;
  accountKey: string;
  limit: number;
  totalUsage: number;
  spentThisMonth: number;
  estimatedCashback: number;
  lifetimeCashback: number;
  totalFees: number;
  netEarn: number;
  transactionCount: number;
  history: CardHistory[];
}

export interface BankAccountSummary {
  name: string;
  limit: number;
  totalUsage: number;
  utilization: number;
  remainingLimit: number;
  isShared: boolean;
  totalRefund: number;
  totalFees: number;
  totalEarn: number;
  estimatedCashback?: number;
  cards: CardStat[];
}

export interface BankSummary {
  name: string;
  accounts: BankAccountSummary[];
}
