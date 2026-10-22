import type { DataSource } from './source';

export interface Ticker {
  symbol: string;
  type: string;
  provider: string;
  currency?: string;
  price?: number;
  timestamp?: string;
  source?: DataSource;
}

/** What a rate means: sources in one fallback chain quote different rates for the same pair. */
export type RateBasis = 'bank_sell' | 'market' | 'reference' | 'crypto_market';

export interface ExchangeRate {
  from: string;
  to: string;
  rate: number;
  provider: string;
  timestamp?: string;
  /** Missing on copies cached before the server sent it. */
  basis?: RateBasis;
  source?: DataSource;
}

export interface BankRate {
  bank: string;
  tenor: string;
  rate: number;
}

/** Published savings rates with where they came from. */
export interface BankRates {
  rates: BankRate[];
  source?: DataSource;
}

/** A flexible savings product: an e-wallet earning balance, spendable and withdrawable any time. */
export interface FlexibleSaving {
  id: string;
  provider: string;
  product: string;
  /** Percent a year. */
  rate: number;
  /** business_cooperation is not a bank deposit and carries financial risk. */
  backing: 'deposit' | 'business_cooperation';
  /** Empty = pays from the same day. */
  earningStarts: 'next_working_day' | '';
  withdrawalMaxWorkingHours?: number;
  currency?: string;
  updatedAt?: string;
  url: string;
  asOf?: string;
  stale?: boolean;
}

/** GET /api/market/flexible-savings: the products that loaded, and the providers that failed. */
export interface FlexibleSavings {
  products: FlexibleSaving[];
  failed: string[];
  source?: DataSource;
}
