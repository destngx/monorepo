/**
 * Investments Feature - Types
 */

export interface Investment {
  symbol: string;
  quantity: number;
  buyPrice: number;
  currentPrice: number;
  currency: string;
}

export interface PortfolioAnalysis {
  totalValue: number;
  gainLoss: number;
  gainLossPercent: number;
  assets: Investment[];
}

export interface GoldExtra {
  rateUsdToVnd: number;
}

export interface UsdHistoryRaw {
  reportDate: string | number;
  rateSellUsd: number;
}

export interface UsdHistoryItem {
  date: string;
  price: number;
  rawDate: number;
}

export interface GoldHistoryRaw {
  reportDate: string | number;
  askSjc: number;
  bidSjc: number;
  price: number;
}

export interface GoldHistoryItem {
  date: string;
  vnAsk: number;
  vnBid: number;
  world: number;
  rawDate: number;
}

export interface FmarketFund {
  id: number;
  name: string;
  symbol: string;
  code?: string;
  nav: number;
  type: string;
  managementFee: number;
  expectedReturn: number;
  [key: string]: any;
}

export interface FmarketProductDetails {
  id: number;
  code: string;
  name: string;
  description?: string;
  fundAssetType: 'STOCK_FUND' | 'BOND_FUND' | 'BALANCED_FUND' | 'MONEY_MARKET_FUND';
  owner?: {
    name: string;
    shortName: string;
  };
  [key: string]: any;
}
