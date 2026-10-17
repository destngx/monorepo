import type { DepositRate } from './investments';
import type { IdleCashSettings } from './preferences';

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

/** Why an account is left out of ready money. */
export type ExclusionReason = 'crypto' | 'not_ready';

export interface ExcludedAccount {
  name: string;
  balance: number;
  reason: ExclusionReason;
}

/** The safe-to-spend setting from the Preferences tab, with the ETag a change must send. */
export interface SafeToSpendReadiness {
  notReadyAccounts: string[];
  etag: string;
}

/**
 * Safe to spend until the next wage (D33): readyMoney - billsDue - cardPayments.
 * Bills and card payments count when unpaid and due before the next income
 * (overdue ones too; one due on payday is paid by that wage), else in the
 * next 30 days inclusive.
 */
export interface SafeToSpend {
  asOf: string;
  until: string;
  untilIncome: boolean;
  incomeName?: string;
  days: number;
  /** Positive balances of active, rarely used, bank and cash accounts, without crypto or not-ready accounts. */
  readyMoney: number;
  readyAccounts: { name: string; balance: number }[];
  excludedAccounts: ExcludedAccount[];
  readiness: SafeToSpendReadiness;
  billsDue: number;
  cardPayments: number;
  /** Can be negative; safeToSpend floors it at 0 and shortBy is the shortfall. */
  amount: number;
  safeToSpend: number;
  shortBy: number;
  perDay: number;
  counted: BillOccurrence[];
  explanation: string;
  readyRule: string;
  /** Marked names that are no longer accounts (renamed or deleted); ignored. */
  unknownNotReady: string[];
  issues: string[];
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
  readyMoney: number;
  /** Unpaid bills and card payments counted before the horizon. */
  dueBeforeUntil: number;
  excludedAccounts: number;
  until: string;
  untilIncome: boolean;
  suggestions: number;
}

/** POST /api/bills/safe-to-spend/accounts body. */
export interface AccountReadinessInput {
  account: string;
  ready: boolean;
  etag: string;
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

/** idle = max(0, readyMoney - keepLiquid - dayToDayFloat) (D34). */
export interface IdleCashComponents {
  /** Safe to spend's ready money (D33) without the emergency accounts. */
  readyMoney: number;
  billsDue: number;
  cardPayments: number;
  goalSetAside: number;
  /** The emergency fund target. */
  emergencyFund: number;
  /** False when the target comes from emergency months x monthly needs. */
  emergencyFromGoal: boolean;
  /** Held in the emergency accounts. */
  emergencyInAccounts: number;
  /** min(emergencyInAccounts, emergencyFund). */
  emergencyCovered: number;
  /** The rest of the target, kept liquid from ready money. */
  emergencyFromReady: number;
  /** billsDue + cardPayments + goalSetAside + emergencyFromReady. */
  keepLiquid: number;
  /** Day-to-day money kept ready: the preference, else bufferMonths x averageMonthlyExpense. */
  dayToDayFloat: number;
  floatFromPreference: boolean;
  bufferMonths: number;
  averageMonthlyExpense: number;
}

/** A ready (or emergency) account, its rate (from its note, else 0) and the idle part assumed in it. */
export interface IdleCashAccount {
  name: string;
  balance: number;
  rate: number;
  rateFromNote: boolean;
  idle: number;
}

/** Rolling 1-month deposits opened spacingDays apart. */
export interface IdleCashLadder {
  parts: number;
  partAmount: number;
  spacingDays: number;
}

/** The emergency fund held in emergency accounts (D34). */
export interface IdleCashEmergency {
  /** Listed whatever the balance, so they can be unmarked. */
  accounts: IdleCashAccount[];
  balance: number;
  target: number;
  currentRate: number;
  rateGapPoints: number;
  extraPerYear: number;
  /** The best instant-access rate beats them by the thresholds. */
  betterRate: boolean;
  settings: IdleCashSettings;
  /** Marked names that are no longer accounts (ignored). */
  unknown: string[];
}

/** GET /api/bills/idle-cash */
export interface IdleCash {
  asOf: string;
  idle: number;
  components: IdleCashComponents;
  /** Everyday ready accounts, lowest rate first. */
  accounts: IdleCashAccount[];
  /** Crypto and not-ready accounts, never idle. */
  excluded: ExcludedAccount[];
  emergency: IdleCashEmergency;
  /** Weighted rate of the idle cash, percent a year. */
  currentRate: number;
  /** The best instant-access rate (no-term or 1-month), a proxy; null when none is published. */
  best: DepositRate | null;
  ratesIssue?: string;
  /** Set when best is a 1-month deposit and the idle cash is big enough to split. */
  ladder: IdleCashLadder | null;
  rateGapPoints: number;
  extraPerYear: number;
  verdict: IdleCashVerdict;
  reasons: string[];
  issues: string[];
  minRateGapPoints: number;
  minExtraPerYear: number;
  keepLiquidRule: string;
  rateRule: string;
  thresholdRule: string;
}

/** POST /api/bills/idle-cash/emergency-accounts body. */
export interface EmergencyAccountInput {
  account: string;
  emergency: boolean;
  etag: string;
}
