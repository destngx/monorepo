import type { Transaction } from './transactions';

/** Inclusive range of merchant category codes; a single code has from === to. */
export interface MccRange {
  from: number;
  to: number;
}

/** The period a programme version counts caps and spend tiers in (C7). */
export type CapPeriod = 'cycle' | 'month';

/**
 * Pays `rate` percent on payments whose MCC is in `mccs`, up to `cap` per
 * period, when the payment is at least `minTransaction` and the period's
 * qualifying spend at least `minCycleSpend` (WM-RULE-7).
 */
export interface CashbackRule {
  name: string;
  mccs: MccRange[];
  rate: number;
  /** Per-period cap; 0 means none. */
  cap: number;
  minTransaction: number;
  minCycleSpend: number;
  /** Applies to spend no other rule matches, including unknown MCCs. */
  fallback?: boolean;
  /** Earns nothing on what it matches, ahead of every other rule. */
  exclude?: boolean;
  /** Rules naming the same group share one cap: what they earn together is bounded by `cap`. */
  capGroup?: string;
}

/** A full copy of a programme's terms from a date (C2). */
export interface ProgramVersion {
  from: string;
  annualFee: number;
  /** Card-level cap per period; 0 means none. */
  cap: number;
  period: CapPeriod;
  rules: CashbackRule[];
}

export interface Program {
  name: string;
  bank: string;
  versions: ProgramVersion[];
}

export interface Bank {
  name: string;
  sharedAccount: string;
  referenceLimit: number;
}

/** A card the user holds (Cards tab). */
export interface CardHolding {
  name: string;
  program: string;
  keyword: string;
  expiry: string;
  statementDay: number;
  dueDay: number;
  default?: boolean;
  /** First day the card is no longer used; its history stays. */
  closedFrom?: string;
  /** The card number's last 4 digits; Smart Capture matches them on materials. */
  last4?: string;
}

export type MccStatus = 'confirmed' | 'assumed' | 'unknown';

/** Payees containing `pattern` take `mcc` from `from` on (absent = always). */
export interface MerchantMcc {
  pattern: string;
  mcc: number;
  from?: string;
  confidence: Exclude<MccStatus, 'unknown'>;
  note?: string;
}

/** The sheet's card configuration (GET /cards/programs, WM-CARD-3). */
export interface CardConfig {
  programs: Program[];
  banks: Bank[];
  cards: CardHolding[];
  merchants: MerchantMcc[];
}

/** Cashback split into the part every MCC involved is confirmed for and the estimated rest (C5). */
export interface CashbackAmount {
  confirmed: number;
  estimated: number;
  total: number;
}

export interface CardCycle {
  statementDay: number;
  dueDay: number;
  cycleStart: string;
  cycleEnd: string;
  cycleDays: number;
  cycleElapsedDays: number;
  cycleProgress: number;
  /** The 29th (28th in February) the statement must be paid by (C8). */
  payBy: string;
  daysUntilPayBy: number;
  /** Within five days of pay-by with a statement still to pay. */
  payByUrgent: boolean;
}

/**
 * The statement the next pay-by settles: the card's spend, fees and cashback
 * refunds in that statement cycle. Payments already made are not deducted;
 * an open cycle can still grow (WM-CARD-15).
 */
export interface CardStatement {
  start: string;
  /** Inclusive: the day before the next statement day. */
  end: string;
  open?: boolean;
  spent: number;
  fees: number;
  cashback: number;
  amount: number;
  /** The cashback credits behind `cashback`. */
  refunds?: StatementRefund[];
}

/**
 * A cashback credit on a statement: it lowers the statement it is credited
 * in, while `for` (the month in its memo) is the period it was earned in.
 */
export interface StatementRefund {
  date: string;
  for: string;
  amount: number;
}

/** open: not ended; awaiting: no refund yet; below/above: the refund is outside confirmed..total (WM-CARD-11). */
export type ReconciliationStatus = 'open' | 'awaiting' | 'match' | 'below' | 'above';

/** One counting period; `month` is the month it starts in. */
export interface CardPeriodHistory {
  month: string;
  start?: string;
  end?: string;
  spent: number;
  cashback: CashbackAmount;
  refund: number;
  status: ReconciliationStatus;
  efficiency: number;
  missed: number;
}

export interface RuleStanding {
  /** The rule's name, or its cap group's. */
  name: string;
  /** A cap group's rules; absent for a single rule. */
  rules?: string[];
  earned: CashbackAmount;
  cap: number;
  capLeft: number;
}

export interface NextTier {
  minCycleSpend: number;
  spendNeeded: number;
  rules: string[];
}

/** A card's position in its current counting period (WM-CARD-12). */
export interface PeriodStanding {
  start: string;
  end: string;
  spent: number;
  qualifyingSpend: number;
  cashback: CashbackAmount;
  cap: number;
  capLeft: number;
  rules: RuleStanding[];
  nextTier?: NextTier;
  paymentCount: number;
  unconfirmedPayments: number;
}

export interface PaymentMcc {
  code?: number;
  status: MccStatus;
}

export interface CardTransaction {
  transaction: Transaction;
  mcc: PaymentMcc;
  rule?: string;
  /** Cashback before period caps. */
  cashback: number;
}

export type UtilizationBand = 'good' | 'fair' | 'high';

export interface CardReport {
  name: string;
  bank: string;
  program: string;
  keyword: string;
  accountName: string;
  expiry: string;
  last4?: string;
  /** Closing date when the card is closed: it has no current period or pay-by. */
  closedFrom?: string;
  /** Start of the programme version in effect today (a closed card: its last); absent when none is. */
  versionFrom?: string;
  annualFee: number;
  cap: number;
  capPeriod?: CapPeriod;
  rules: CashbackRule[];
  cycle: CardCycle;
  /** Absent for a closed card. */
  statement?: CardStatement;
  standing: PeriodStanding;
  refundYtd: number;
  /** This year's fee rows (memo "bankfee"), net of any given back; net earn counts the fee only once paid. */
  feesPaidYtd: number;
  netEarnYtd: number;
  lifetimeFees: number;
  history: CardPeriodHistory[];
  /** The current period's transactions, newest first. */
  transactions: CardTransaction[];
}

/** A bank's shared limit and its held cards (WM-CARD-16). */
export interface CardBankReport {
  name: string;
  sharedAccount: string;
  limit: number;
  /** What the accounts its cards bill to owe; money paid in ahead counts as room. */
  usage: number;
  utilization: number;
  utilizationBand: UtilizationBand;
  remainingLimit: number;
  cashback: CashbackAmount;
  refundYtd: number;
  feesPaidYtd: number;
  netEarnYtd: number;
  cards: CardReport[];
}

/** What one held card would earn on a payment now (WM-CARD-13). */
export interface CardSuggestion {
  card: string;
  bank: string;
  program: string;
  mcc: PaymentMcc;
  rule?: string;
  cashback: CashbackAmount;
  fitsLimit: boolean;
  remainingLimit: number;
}

export interface CardPaymentQuery {
  payee: string;
  amount: number;
  mcc?: number;
}

export interface ProgramComparison {
  program: string;
  bank: string;
  held: boolean;
  versionFrom: string;
  spent: number;
  cashback: CashbackAmount;
  annualFee: number;
  net: number;
}

/** Programmes replayed over the last year of card payments, best net first (WM-CARD-14). */
export interface ProgramComparisonReport {
  from: string;
  to: string;
  payments: number;
  programs: ProgramComparison[];
}

/** A Cards row to write (POST /cards, PATCH /cards/:name; WM-FILL-12). */
export interface CardInput {
  name: string;
  program: string;
  keyword: string;
  expiry?: string;
  statementDay: number;
  dueDay: number;
  default?: boolean;
  closedFrom?: string;
  last4?: string;
  /** The bank's shared account credit limit (Accounts C); absent leaves it. */
  limit?: number;
}

export interface ProgramRuleInput {
  rule: string;
  /** Codes and ranges as text, e.g. "5411, 5812-5814"; blank matches any. */
  mccs?: string;
  rate: number;
  cap?: number;
  minTransaction?: number;
  minCycleSpend?: number;
  fallback?: boolean;
  exclude?: boolean;
  capGroup?: string;
}

/** A programme version with its rules (POST /cards/programs; WM-FILL-13). */
export interface ProgramInput {
  program: string;
  bank: string;
  /** For a bank with no Banks row: the account its cards bill to. */
  sharedAccount?: string;
  from: string;
  annualFee?: number;
  cap?: number;
  capPeriod: CapPeriod;
  rules: ProgramRuleInput[];
}

/** A payee MCC to save (POST /cards/merchants; WM-FILL-14). */
export interface MerchantInput {
  pattern: string;
  mcc: number;
  from?: string;
  confidence?: Exclude<MccStatus, 'unknown'>;
  note?: string;
}
