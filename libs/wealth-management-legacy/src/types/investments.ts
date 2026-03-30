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
