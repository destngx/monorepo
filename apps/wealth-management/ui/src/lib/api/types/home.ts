// The home inbox (ROADMAP R5, WM-HOME-1): GET /api/home/needs-you.

export type NeedsYouKind =
  | 'drafts'
  | 'bill-overdue'
  | 'card-due'
  | 'short-before-payday'
  | 'tight-before-payday'
  | 'minimum-balance'
  | 'budget-over-pace'
  | 'lent-overdue'
  | 'goal-plan-short'
  | 'goal-behind'
  | 'rebalance'
  | 'uncategorized'
  | 'recommendation'
  | 'idle-cash'
  | 'emergency-fund-rate'
  | 'month-close';

/** Shared status token names (lib/status.ts). */
export type NeedsYouStatus = 'critical' | 'serious' | 'warning' | 'info';

export interface NeedsYouItem {
  id: string;
  kind: NeedsYouKind;
  status: NeedsYouStatus;
  /** No names of people or goals. */
  headline: string;
  detail?: string;
  /** The money the item is about; null when none. */
  amountVnd: number | null;
  /** Estimated VND a year; null when not known. */
  impactVnd: number | null;
  /** yyyy-mm-dd when it is due or runs out. */
  dueDate?: string;
  /** How many things the item groups. */
  count?: number;
  /** App route to act on it, with its query. */
  route: string;
  recommendationId?: string;
}

export type NeedsYouPart =
  | 'drafts'
  | 'bills'
  | 'budget'
  | 'moneyLent'
  | 'goals'
  | 'allocation'
  | 'transactions'
  | 'recommendations'
  | 'idleCash'
  | 'close';

export interface NeedsYou {
  asOf: string;
  /** Ranked: severity, then impact or amount, then due date. */
  items: NeedsYouItem[];
  /** Sources that failed to load: their items are missing. */
  unavailable: { part: NeedsYouPart; label: string }[];
  ranking: string;
}
