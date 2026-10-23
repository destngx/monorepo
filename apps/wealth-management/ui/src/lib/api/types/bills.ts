import type { DepositRate } from './investments';
import type { AccountMinimumSettings, IdleCashSettings } from './preferences';

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
  /** Days after its date an unpaid occurrence stays due before it is overdue (D42). */
  graceDays?: number;
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
  /** yyyy-mm-dd: the last day an unpaid occurrence is still due, when its item has grace days. */
  graceUntil?: string;
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

/** An account counted in ready money: only the balance above its minimum (D35) is spendable. */
export interface SpendableAccount {
  name: string;
  balance: number;
  /** min(balance, minimum); 0 without a minimum. */
  reserved: number;
  /** balance - reserved: what the account adds to ready money. */
  spendable: number;
}

/** Whether an account reaches its minimum average balance this month at today's balance. */
export type MinimumBalanceStatus = 'on_track' | 'at_risk';

/** One account's minimum balance and this month's average daily balance against it (D35). */
export interface AccountMinimumBalance {
  account: string;
  minimum: number;
  note?: string;
  balance: number;
  reserved: number;
  /** yyyy-mm and its last day. */
  month: string;
  monthEnd: string;
  daysInMonth: number;
  /** Finished days (1st to yesterday). */
  daysCounted: number;
  /** Days left, today included. */
  daysLeft: number;
  sumSoFar: number;
  averageSoFar: number;
  /** The month's average if the balance stays as it is today. */
  projectedAverage: number;
  /** The balance to hold from today to the month end (0 when covered). */
  requiredBalance: number;
  extraNeeded: number;
  status: MinimumBalanceStatus;
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
/** Where a set-aside amount comes from (D36): the monthly budget limit, or the average of the last 6 completed months. */
export type SetAsideBasis = 'budget' | 'average';

/** A fixed category's cost still to pay in one month of the window: expected - spent - inBills, floored at 0. */
export interface FixedCost {
  category: string;
  /** yyyy-mm */
  month: string;
  expected: number;
  basis: SetAsideBasis;
  spent: number;
  /** Its unpaid recurring bills that month (counted with the bills, or paid by the wage). */
  inBills: number;
  amount: number;
}

/** An essential category's spending until the horizon: perDay x days. */
export interface EssentialNeed {
  category: string;
  perDay: number;
  basis: SetAsideBasis;
  days: number;
  amount: number;
}

/** Safe-to-spend in tiers (D33, D36): ready money - committed - essentials = free to spend (safeToSpend). */
export interface SafeToSpend {
  asOf: string;
  until: string;
  untilIncome: boolean;
  incomeName?: string;
  days: number;
  /** Positive balances of active, rarely used, bank and cash accounts above their minimums, without crypto or not-ready accounts. */
  readyMoney: number;
  readyAccounts: SpendableAccount[];
  excludedAccounts: ExcludedAccount[];
  readiness: SafeToSpendReadiness;
  /** The minimum balances kept in the ready accounts (D35). */
  reserved: number;
  /** The saved minimums, with the ETag a change must send. */
  minimumBalance: AccountMinimumSettings;
  /** Every account with a minimum, ready or not. */
  minimumBalances: AccountMinimumBalance[];
  minimumRule: string;
  /** Names with a minimum that are no longer accounts; ignored. */
  unknownMinimum: string[];
  billsDue: number;
  cardPayments: number;
  /** Fixed categories' costs still to pay. */
  fixedCosts: number;
  fixed: FixedCost[];
  /** billsDue + cardPayments + fixedCosts: never spendable. */
  committed: number;
  /** readyMoney - committed; can be negative (shortBy is the shortfall). */
  amount: number;
  /** Essential categories' spending until the horizon: money for needs only. */
  essentials: number;
  essential: EssentialNeed[];
  /** Free to spend: amount - essentials, floored at 0. */
  safeToSpend: number;
  shortBy: number;
  /** Essentials not covered once committed money is kept. */
  tightBy: number;
  /** safeToSpend over days. */
  perDay: number;
  /** False when no expense category is tagged fixed or essential: nothing is set aside. */
  natureTagged: boolean;
  counted: BillOccurrence[];
  explanation: string;
  readyRule: string;
  setAsideRule: string;
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
  /** Free to spend. */
  safeToSpend: number;
  shortBy: number;
  tightBy: number;
  readyMoney: number;
  /** Unpaid bills and card payments counted before the horizon. */
  dueBeforeUntil: number;
  fixedCosts: number;
  /** dueBeforeUntil + fixedCosts. */
  committed: number;
  essentials: number;
  excludedAccounts: number;
  minimumReserved: number;
  minimumBalanceAccounts: number;
  minimumBalanceAtRisk: number;
  minimumBalanceExtraNeeded: number;
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

/** POST /api/bills/safe-to-spend/minimum-balance body: a null or 0 minimum removes it. */
export interface AccountMinimumInput {
  account: string;
  minimum: number | null;
  note?: string;
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
  graceDays?: number;
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
  /** Safe to spend's ready money (D33) without the emergency accounts, above each minimum (D35). */
  readyMoney: number;
  /** Minimum balances kept in the everyday ready accounts (D35), never idle. */
  minimumReserved: number;
  billsDue: number;
  cardPayments: number;
  goalSetAside: number;
  /** The emergency fund target. */
  emergencyFund: number;
  /** False when the target comes from emergency months x monthly needs. */
  emergencyFromGoal: boolean;
  /** True when no amount is typed, so the target is emergencyMonths x needsPerMonth. */
  emergencyTargetComputed: boolean;
  emergencyMonths: number;
  needsPerMonth: number;
  /** No category is tagged fixed or essential, so all spending stands in for needs. */
  needsFromTotalExpense: boolean;
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
  /** The minimum balance kept (D35): never idle, never moved. */
  reserved: number;
  rate: number;
  rateFromNote: boolean;
  idle: number;
  /** The linked product's display name, set when the rate is its live rate. */
  product?: string;
  productId?: string;
}

/** Rolling 1-month deposits opened spacingDays apart. */
export interface IdleCashLadder {
  parts: number;
  partAmount: number;
  spacingDays: number;
}

/** One day to start a move: money in transit earns nothing. */
export interface SavingsMoveDay {
  date: string;
  weekday: string;
  daysWithoutInterest: number;
  interestLost: number;
  /** Days until the extra interest earns back the interest lost. */
  paybackDays: number;
}

/** Moving money from an account to a better instant-access place, with the best day to do it. */
export interface SavingsMove {
  from: string;
  fromRate: number;
  fromProduct?: string;
  amount: number;
  /** Already a place phrase. */
  to: string;
  toRate: number;
  /** The new place's backing when it is a flexible product. */
  toBacking?: 'deposit' | 'business_cooperation';
  rateGapPoints: number;
  extraPerMonth: number;
  extraPerYear: number;
  today: SavingsMoveDay;
  bestDay: SavingsMoveDay;
  worstCaseExtraDays: number;
  /** Ordered strategy sentences; they contain VND amounts. */
  steps: string[];
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
  /** Emergency accounts to the best place, when betterRate. */
  moves: SavingsMove[];
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
  /** Every instant-access place compared, best first. */
  options: DepositRate[];
  /** The idle part of each account to the best place, only when the verdict is worth_moving. */
  moves: SavingsMove[];
  /** A linked wallet account's day-to-day money to a better wallet product. */
  switches: SavingsMove[];
  /** Why products are missing or partial. */
  productsIssue?: string;
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
  productRule: string;
  thresholdRule: string;
}

/** POST /api/bills/idle-cash/emergency-accounts body. */
export interface EmergencyAccountInput {
  account: string;
  emergency: boolean;
  etag: string;
}

/** POST /api/bills/idle-cash/account-products body; an empty product unlinks. */
export interface AccountProductInput {
  account: string;
  product: string;
  etag: string;
}
