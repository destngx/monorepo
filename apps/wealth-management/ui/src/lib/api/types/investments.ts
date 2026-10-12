import type { Currency } from './accounts';
import type { ExchangeRate } from './rates';

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
