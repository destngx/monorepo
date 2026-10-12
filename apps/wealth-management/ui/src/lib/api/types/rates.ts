export interface Ticker {
  symbol: string;
  type: string;
  provider: string;
  currency?: string;
  price?: number;
  timestamp?: string;
}

export interface ExchangeRate {
  from: string;
  to: string;
  rate: number;
  provider: string;
  timestamp?: string;
}

export interface BankRate {
  bank: string;
  tenor: string;
  rate: number;
}
