import type { DepositRate } from './investments';

/** Recurring tab Kind column (B). */
export type RecurringKind = 'income' | 'bill' | 'subscription' | 'insurance' | 'card';
export type RecurringFrequency = 'monthly' | 'quarterly' | 'yearly';
export type RecurringStatus = 'active' | 'paused';
export type BillStatus = 'paid' | 'due' | 'overdue';
export type SuggestionConfidence = 'high' | 'medium' | 'low';

/** One row of the app-managed Recurring tab. */
export interface RecurringItem {
  /** recurring-N; with etag it addresses the row for edits. */
  id: string;
  row: number;
  name: string;
  kind: RecurringKind;
  /** null only on card rows, which use the statement. */
  amount: number | null;
  frequency: RecurringFrequency;
  /** Day of month; 0 on card rows. */
  day: number;
  /** 1-12 for quarterly (first month) and yearly items. */
  month?: number;
  account?: string;
  category?: string;
  match?: string;
  status: RecurringStatus;
  note?: string;
  etag: string;
}

export interface BillMatch {
  transactionId: string;
  date: string;
  amount: number;
  payee: string;
  account: string;
}

/** One expected payment (or income) on the calendar. */
export interface BillOccurrence {
  itemId?: string;
  name: string;
  kind: RecurringKind;
  source: 'recurring' | 'card';
  /** yyyy-mm-dd */
  expectedDate: string;
  expectedAmount: number;
  status: BillStatus;
  daysUntil: number;
  account?: string;
  note?: string;
  paidAmount: number;
  paidDate?: string;
  matches?: BillMatch[];
  remaining: number;
  /** The amount can still change (open card statement, card without settings). */
  estimate?: boolean;
}

export interface BillItem extends RecurringItem {
  current: BillOccurrence | null;
  nextDate?: string;
  lastMatch: BillMatch | null;
  matchCount: number;
}

export interface RecurringSuggestion {
  key: string;
  name: string;
  kind: RecurringKind;
  amount: number;
  frequency: RecurringFrequency;
  day: number;
  month?: number;
  account?: string;
  category: string;
  match: string;
  /** yyyy-mm */
  lastSeen: string;
  seen: number;
  checked: number;
  amountSpread: number;
  daySpread: number;
  confidence: SuggestionConfidence;
}

export interface BillCalendar {
  from: string;
  to: string;
  occurrences: BillOccurrence[];
  upcomingOut: number;
  upcomingCount: number;
  upcomingIn: number;
  overdueCount: number;
  overdueAmount: number;
  paidOut: number;
}

export interface SafeToSpend {
  asOf: string;
  until: string;
  untilIncome: boolean;
  incomeName?: string;
  days: number;
  spendableCash: number;
  cashAccounts: { name: string; balance: number }[];
  billsDue: number;
  cardPayments: number;
  goalSetAside: number;
  goalPlanUsed: boolean;
  /** Can be negative; safeToSpend floors it at 0 and shortBy is the shortfall. */
  amount: number;
  safeToSpend: number;
  shortBy: number;
  perDay: number;
  counted: BillOccurrence[];
  explanation: string;
  goalSetAsideRule: string;
}

export interface RecurringIssue {
  row: number;
  message: string;
}

/** GET /api/bills */
export interface Bills {
  asOf: string;
  tabExists: boolean;
  items: BillItem[];
  issues: RecurringIssue[];
  suggestions: RecurringSuggestion[];
  calendar: BillCalendar;
  safeToSpend: SafeToSpend;
}

export interface BillsFacts {
  items: number;
  upcoming30Days: number;
  upcomingCount: number;
  overdueCount: number;
  overdueAmount: number;
  safeToSpend: number;
  shortBy: number;
  until: string;
  untilIncome: boolean;
  suggestions: number;
}

/** GET /api/bills/safe-to-spend: the home page headline (ROADMAP R5). */
export interface BillsSummary {
  safeToSpend: SafeToSpend;
  facts: BillsFacts;
  overdue: BillOccurrence[];
  dueSoon: BillOccurrence[];
}

/** POST /api/bills/items and PATCH /api/bills/items/:id body. */
export interface RecurringInput {
  name: string;
  kind: RecurringKind;
  amount?: number;
  frequency?: RecurringFrequency;
  day?: number;
  month?: number;
  account?: string;
  category?: string;
  match?: string;
  status?: RecurringStatus;
  note?: string;
}

/** POST /api/bills/suggestions/confirm body: blank fields keep the suggested values. */
export interface RecurringConfirmInput {
  key: string;
  name?: string;
  kind?: RecurringKind;
  amount?: number;
  day?: number;
  account?: string;
  category?: string;
  match?: string;
  status?: RecurringStatus;
  note?: string;
}

/** Idle cash verdict (ROADMAP R6). */
export type IdleCashVerdict = 'worth_moving' | 'not_worth_it' | 'nothing_idle' | 'no_rate';

/** idle = max(0, spendableCash - keepLiquid). */
export interface IdleCashComponents {
  spendableCash: number;
  billsDue: number;
  cardPayments: number;
  goalSetAside: number;
  spendingBuffer: number;
  bufferMonths: number;
  averageMonthlyExpense: number;
  emergencyFund: number;
  /** False when the target comes from emergency months x monthly needs. */
  emergencyFromGoal: boolean;
  keepLiquid: number;
}

/** A spendable cash account, its rate (from its note, else 0) and the idle part assumed in it. */
export interface IdleCashAccount {
  name: string;
  balance: number;
  rate: number;
  rateFromNote: boolean;
  idle: number;
}

/** GET /api/bills/idle-cash */
export interface IdleCash {
  asOf: string;
  idle: number;
  components: IdleCashComponents;
  /** Lowest rate first. */
  accounts: IdleCashAccount[];
  /** Weighted rate of the idle cash, percent a year. */
  currentRate: number;
  /** The comparison rate, a proxy; null when unavailable. */
  best: DepositRate | null;
  ratesIssue?: string;
  rateGapPoints: number;
  extraPerYear: number;
  verdict: IdleCashVerdict;
  reasons: string[];
  minRateGapPoints: number;
  minExtraPerYear: number;
  keepLiquidRule: string;
  rateRule: string;
  thresholdRule: string;
}
