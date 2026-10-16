// Monthly close and net worth history (ROADMAP R8, P3; domain/monthly_close.go, net_worth_history.go).

export type CloseItemKey = 'drafts' | 'categorize' | 'budget' | 'goals' | 'moneyLent' | 'recommendations' | 'snapshot';
export type CloseItemState = 'done' | 'action' | 'unavailable';

/** One checklist line; detail never carries amounts (they stay masked in the UI). */
export interface CloseItem {
  key: CloseItemKey;
  title: string;
  state: CloseItemState;
  detail: string;
  count: number | null;
  amount: number | null;
  target: number | null;
}

/** One row of the Snapshots tab; only month and netWorth are required. */
export interface MonthSnapshot {
  month: string;
  asOf: string;
  netWorth: number;
  assets: number | null;
  liabilities: number | null;
  moneyLent: number | null;
  cash: number | null;
  funds: number | null;
  crypto: number | null;
  income: number | null;
  expense: number | null;
  savings: number | null;
  savingsRate: number | null;
  healthScore: number | null;
  savingsScore: number | null;
  emergencyScore: number | null;
  debtScore: number | null;
  diversificationScore: number | null;
  emergencyMonths: number | null;
  writtenOn: string;
  note: string;
  row: number;
}

export interface MonthlyClose {
  month: string;
  asOf: string;
  monthEnded: boolean;
  canWriteSnapshot: boolean;
  items: CloseItem[];
  done: number;
  checked: number;
  snapshot: MonthSnapshot | null;
}

export interface SnapshotSaveResult {
  snapshot: MonthSnapshot;
  created: boolean;
}

export type NetWorthSource = 'snapshot' | 'estimated' | 'live';

export interface NetWorthPoint {
  month: string;
  asOf: string;
  netWorth: number;
  source: NetWorthSource;
  estimated: boolean;
  change: number | null;
  /** Income - expense between this point and the previous one; null unless both are measured. */
  savings: number | null;
  /** The rest of the change: price moves, corrections, anything not recorded as income or expense. */
  market: number | null;
}

export interface NetWorthMilestone {
  month: string;
  kind: 'threshold' | 'all_time_high';
  threshold: number | null;
  netWorth: number;
  label: string;
  estimated: boolean;
}

export interface SnapshotIssue {
  row: number;
  message: string;
}

export interface NetWorthHistory {
  asOf: string;
  current: number;
  points: NetWorthPoint[];
  milestones: NetWorthMilestone[];
  snapshots: number;
  estimated: number;
  tabExists: boolean;
  issues: SnapshotIssue[];
}
