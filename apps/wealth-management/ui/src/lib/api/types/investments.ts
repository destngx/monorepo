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
