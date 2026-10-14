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
