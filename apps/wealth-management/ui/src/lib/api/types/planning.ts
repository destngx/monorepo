import type { Transaction } from './transactions';

/** Goal Plan status column (F). */
export type GoalStatus = 'active' | 'paused' | 'done';

/** A goal's standing in the plan, computed by the engine. */
export type GoalPace = 'done' | 'paused' | 'funded' | 'on_track' | 'behind' | 'no_deadline';

/** One row of the Goal Plan tab. */
export interface PlannedGoal {
  /** goal-N; with etag it addresses the row for edits. */
  id: string;
  row: number;
  name: string;
  /** null only on the emergency fund, whose target is then computed. */
  target: number | null;
  /** yyyy-mm; absent means no deadline. */
  targetDate?: string;
  priority: number;
  startingAmount: number;
  status: GoalStatus;
  note?: string;
  /** The goal may draw on slow money (column H); off by default. */
  useSlowMoney: boolean;
  etag: string;
}

/** A goal in the computed plan (D31, D32). */
export interface GoalProgress extends PlannedGoal {
  emergencyFund: boolean;
  targetAmount: number;
  targetComputed: boolean;
  funded: number;
  fromPool: number;
  /** The part of fromPool taken from slow money. */
  fromSlow: number;
  remaining: number;
  percent: number;
  monthsLeft: number | null;
  requiredPerMonth: number | null;
  /** yyyy-mm; null when not reachable within 50 years, paused or done. */
  projectedFinish: string | null;
  monthsToReach: number | null;
  yearsToReach: number | null;
  longHorizon: boolean;
  pace: GoalPace;
}

export type PoolRole = 'asset' | 'cardDebt' | 'crypto' | 'slow' | 'negative' | 'archived' | 'empty';

/**
 * The pooled savings balance: total = max(0, assets - cardDebt - spendingBuffer), ready money only.
 * slow (investment and long holding accounts, accounts marked not ready) is apart: only goals that opt in use it.
 */
export interface GoalPool {
  assets: number;
  cardDebt: number;
  spendingBuffer: number;
  bufferMonths: number;
  averageMonthlyExpense: number;
  total: number;
  slow: number;
  /** Shown as upside, never counted (D32). */
  crypto: number;
  archived: number;
  moneyLent: number;
  /** Negative balances of accounts that are not cards: a ledger gap, counted as 0. */
  negativeBalances: number;
  accounts: { name: string; balance: number; role: PoolRole }[];
}

export interface GoalTradeOff {
  goalId: string;
  name: string;
  targetDate: string;
  /** yyyy-mm that alone closes the gap; null when moving this goal is not enough. */
  moveTargetDateTo: string | null;
  /** Target cut that alone closes the gap; null when not enough. */
  lowerTargetBy: number | null;
}

export type PlanVerdict = 'fits' | 'short' | 'no_deadlines';

export interface GoalPlan {
  asOf: string;
  setupNeeded: boolean;
  /** Old Goals rows a setup would import. */
  legacyGoals: number;
  settings: { emergencyMonths: number; spendingBufferMonths: number };
  pool: GoalPool;
  needs: { perMonth: number; months: number; fromTotalExpense: boolean };
  monthlySavings: { perMonth: number; months: number };
  /** Priority order. */
  goals: GoalProgress[];
  unallocated: number;
  unallocatedSlow: number;
  check: {
    requiredPerMonth: number;
    monthlySavings: number;
    /** > 0 is the monthly shortfall. */
    gap: number;
    verdict: PlanVerdict;
    datedGoals: number;
    tradeOffs: GoalTradeOff[];
  };
  issues: { row: number; message: string }[];
}

export interface GoalInput {
  name: string;
  target?: number;
  targetDate?: string;
  priority?: number;
  startingAmount?: number;
  status?: GoalStatus;
  note?: string;
  useSlowMoney?: boolean;
}

export interface GoalPlanSetupResult {
  created: number;
  imported: number;
  skipped: string[];
}

/** One person's money-lent ledger (WM-LEND-1). */
export interface MoneyLentPerson {
  /** The most recent spelling of the payee. */
  name: string;
  lent: number;
  repaid: number;
  /** lent - repaid: positive is owed to you, negative is owed by you, 0 is settled. */
  outstanding: number;
  /** yyyy-mm-dd of the latest transaction. */
  lastActivity: string;
  /** yyyy-mm-dd the open balance started; null when settled. */
  openSince: string | null;
  daysOutstanding: number | null;
  /** yyyy-mm-dd soonest repayment date of the open loans (WM-LEND-6, WM-LEND-7); null when none has one. */
  repayBy: string | null;
  /** Days from today to repayBy; negative when overdue. */
  daysUntilRepay: number | null;
  /** What is left of the loans due on repayBy. */
  repayByAmount: number;
  /** The loans of a balance owed to you, oldest first; empty when nothing is owed to you. */
  loans: LentLoan[];
  /** Newest first. */
  transactions: Transaction[];
}

/** One loan of a balance owed to you; repayments pay off the oldest loan first (WM-LEND-7). */
export interface LentLoan {
  transactionId: string;
  /** yyyy-mm-dd lending day. */
  date: string;
  /** The lending row's Ref #; null when blank. A repayment with the same Ref # pays this loan first (WM-LEND-9). */
  reference: string | null;
  amount: number;
  repaid: number;
  /** amount - repaid; 0 once repaid. */
  outstanding: number;
  daysOutstanding: number;
  /** yyyy-mm-dd newest repayment date written for the loan; null when none or once repaid. */
  repayBy: string | null;
  /** Days from today to repayBy; negative when overdue. */
  daysUntilRepay: number | null;
  /** The parts of repayment transactions that paid the loan, oldest first (WM-LEND-8). */
  repayments: LoanRepayment[];
}

/** The part of a repayment transaction that paid one loan. */
export interface LoanRepayment {
  transactionId: string;
  /** yyyy-mm-dd repayment day. */
  date: string;
  amount: number;
}

/** Money lent per person from lending-category transactions (WM-LEND-1..3, D30). */
export interface MoneyLent {
  asOf: string;
  owedToYou: number;
  youOwe: number;
  peopleOwing: number;
  peopleYouOwe: number;
  /** Longest open balance owed to you; null when nobody owes you. */
  oldestDaysOutstanding: number | null;
  /** Owed to you first, then owed by you, then settled. */
  people: MoneyLentPerson[];
}
