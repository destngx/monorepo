import type { Currency } from './accounts';
import type { ExchangeRate } from './rates';
import type { DataSource } from './source';

export interface Holding {
  platform: string;
  symbol: string;
  quantity: number | null;
  price: number | null;
  currency: Currency;
  value: number | null;
  valueSource: string;
}

export interface HoldingSection {
  headers: string[];
  holdings: Record<string, unknown>[];
  items: Holding[];
}

export interface InvestmentAssets {
  crypto: HoldingSection;
  funds: HoldingSection;
}

// Engine valuation (WM-INV-2/3/4, WM-RULE-8): every VND figure comes from the engine.
export interface LivePrice {
  price: number;
  currency: Currency;
  provider: string;
  timestamp?: string;
  source?: DataSource;
}

export type PortfolioSection = 'crypto' | 'funds';

export interface ValuedHolding extends Holding {
  section: PortfolioSection;
  livePrice?: LivePrice;
  valueVND: number | null;
  issue?: string;
}

export interface PortfolioTotals {
  vndAccounts: number;
  funds: number;
  vnd: number;
  cryptoAccounts: number;
  cryptoHoldings: number;
  crypto: number;
  total: number;
  unvalued: number;
}

export interface PortfolioValuation {
  holdings: ValuedHolding[];
  rates: ExchangeRate[];
  totals: PortfolioTotals;
  issues: string[];
  generatedAt: string;
}

/** Allocation classes (ROADMAP C4); money lent has no target and stays out of drift. */
export type AllocationClass = 'cash' | 'funds' | 'crypto' | 'moneyLent';
export type TargetClass = Exclude<AllocationClass, 'moneyLent'>;
export type AllocationStatus = 'on_target' | 'over' | 'under' | 'no_target';

export interface AllocationLine {
  class: AllocationClass;
  label: string;
  value: number;
  /** Percent of every allocated asset, money lent included. */
  share: number;
  targeted: boolean;
  /** Percent of the targeted total (cash + funds + crypto); null for money lent. */
  current: number | null;
  target: number | null;
  /** Current minus target, in percentage points. */
  drift: number | null;
  status: AllocationStatus;
}

export type AllocationTargets = Record<TargetClass, number>;

/** The one engine allocation shown by Investments, Accounts and Health. */
export interface Allocation {
  lines: AllocationLine[];
  total: number;
  targetedTotal: number;
  targets: AllocationTargets | null;
  driftThreshold: number;
  rebalance: boolean;
  unvalued: number;
  issues: string[];
  generatedAt: string;
}

// Fund performance vs a bank deposit (ROADMAP M3, WM-INV-21..23).

/** One History entry: units bought (positive) or sold (negative). */
export interface FundTrade {
  date: string;
  units: number;
}

/** A trade priced at the NAV it was matched at; amount is negative for a buy. */
export interface FundCashFlow {
  date: string;
  units: number;
  nav: number;
  navDate: string;
  amount: number;
}

/** Today's best published rate, a labelled proxy for the whole period. */
/** How quickly an instant-access comparison can be reached (idle cash, D34). */
export type DepositAccess = 'no_term' | 'one_month';

export interface DepositRate {
  bank: string;
  tenor: string;
  /** 0 for a no-term deposit; absent when the source gives no tenor. */
  tenorMonths?: number;
  /** counter or online, when the source says which. */
  channel?: string;
  /** Set on idle cash's instant-access comparison. */
  access?: DepositAccess;
  /** "no-term, withdraw any time" or "1-month term, rolls monthly; ...". */
  accessLabel?: string;
  /** Percent a year. */
  rate: number;
  updatedAt?: string;
  basis: string;
}

export type VsDeposit = 'ahead' | 'behind' | 'even';
export type FundCostStatus = 'known' | 'no_history' | 'history_invalid' | 'units_mismatch' | 'nav_unavailable';

/** Percentages are percent; annualisedReturn is the money-weighted return (XIRR). */
export interface FundReturn {
  totalBought: number;
  totalSold: number;
  netInvested: number;
  value: number;
  gain: number;
  simpleReturn: number | null;
  annualisedReturn: number | null;
  firstDate: string;
  asOf: string;
  depositValue: number | null;
  depositGain: number | null;
  differenceVND: number | null;
  /** Annualised return minus the deposit rate, in percentage points. */
  differencePoints: number | null;
  vsDeposit?: VsDeposit;
}

export interface FundHoldingPerformance {
  platform: string;
  symbol: string;
  units: number | null;
  historyUnits: number;
  nav: number | null;
  navDate?: string;
  value: number | null;
  costStatus: FundCostStatus;
  costBasis: number | null;
  unrealisedGain: number | null;
  cashFlows: FundCashFlow[];
  return: FundReturn | null;
  issues: string[];
}

export interface FundCodePerformance {
  symbol: string;
  holdings: number;
  value: number;
  costStatus: FundCostStatus;
  return: FundReturn | null;
}

export interface FundPerformanceTotals {
  holdings: number;
  costKnown: number;
  costUnknown: number;
  value: number;
  valueCostKnown: number;
  return: FundReturn | null;
}

export interface FundPerformance {
  holdings: FundHoldingPerformance[];
  funds: FundCodePerformance[];
  totals: FundPerformanceTotals;
  deposit: DepositRate | null;
  depositIssue?: string;
  generatedAt: string;
}

// My exposure (ROADMAP M1/M2): the engine computes every figure.

/** A price against an earlier one: coins over about 24h, funds since the previous NAV. */
export interface PriceMove {
  basis: '24h' | 'previous_nav';
  price: number;
  previous: number;
  /** Percent. */
  changePct: number;
  asOf: string;
  since: string;
  source?: DataSource;
}

export interface ExposureClass {
  class: AllocationClass;
  label: string;
  value: number;
  /** Percent of the whole allocation. */
  share: number;
  volatile: boolean;
  /** Today's change in VND of the holdings in `covered`; null for a fixed class or without moves. */
  change: number | null;
  changePct: number | null;
  covered: number;
}

/** A coin or fund code summed over every platform. */
export interface HeldPosition {
  symbol: string;
  quantity: number | null;
  price?: LivePrice;
  value: number;
  unvalued: number;
  /** Percent of the section's valued total. */
  share: number;
  move?: PriceMove;
  change: number | null;
  issue?: string;
}

export interface DrawdownScenario {
  fallPct: number;
  loss: number;
  netWorthAfter: number;
  /** Negative percent; null when net worth is not positive. */
  netWorthChangePct: number | null;
}

export interface CryptoExposure {
  value: number;
  holdings: number;
  accounts: number;
  share: number;
  shareOfNetWorth: number | null;
  coins: HeldPosition[];
  change: number | null;
  scenarios: DrawdownScenario[];
}

export interface Exposure {
  total: number;
  netWorth: number;
  volatileValue: number;
  /** Percent of the allocation in funds and crypto. */
  volatileShare: number;
  change: number | null;
  changePct: number | null;
  covered: number;
  classes: ExposureClass[];
  crypto: CryptoExposure;
  funds: HeldPosition[];
  issues: string[];
  generatedAt: string;
}

// Gold and USD as protection (ROADMAP M4).

export interface TrendChange {
  period: string;
  fromDate: string;
  from: number;
  changePct: number;
  /** The history starts after the period start; `from` is the first close. */
  partial: boolean;
  /** What the VND cash would have gained or lost held in the asset instead. */
  onCash: number;
}

export interface DailyClose {
  date: string;
  close: number;
}

export interface HedgeTrend {
  asset: 'gold' | 'usd';
  label: string;
  symbol: string;
  unit: string;
  latest: number;
  latestDate: string;
  changes: TrendChange[];
  points: DailyClose[];
  source?: DataSource;
}

export interface Hedges {
  cash: number;
  cashShare: number;
  vndShare: number;
  gold: HedgeTrend | null;
  usd: HedgeTrend | null;
  issues: string[];
  generatedAt: string;
}
