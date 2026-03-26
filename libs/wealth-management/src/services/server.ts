import { withCache } from '../utils/cache';
import { getAccounts as fetchAccounts } from './sheets/accounts';
import { getBudget as fetchBudget } from './sheets/budget';
import { getCategories as fetchCategories, Category } from './sheets/categories';
import { getExchangeRate as fetchExchangeRate } from './services/exchange-rate-service';
import { getTransactions as fetchTransactions, addTransaction } from './sheets/transactions';
import { getGoals as fetchGoals } from './sheets/goals';
import { getLoans as fetchLoans } from './sheets/loans';
import { getPendingNotifications, markNotificationDone } from './sheets/notifications';
import type { MarketPulseResponse } from './services/market-pulse/types';
import { Account } from '../types/accounts';
import { BudgetItem } from '../types/budget';
import { Transaction } from '../types/transactions';

export * from './index';
export * from './services/investment-service';

// Direct exports from sheets (server-only modules)
export { getPendingNotifications, markNotificationDone };

export interface SearchResult {
  title: string;
  url: string;
  description: string;
}

export interface SearchResponse {
  results: SearchResult[];
}

// Cached service exports
export const getAccounts = withCache('accounts:all', fetchAccounts as () => Promise<Account[]>, 300);
export const getBudget = withCache('budget:all', fetchBudget as () => Promise<BudgetItem[]>, 300);
export const getCategories = withCache('categories:all', fetchCategories as () => Promise<Category[]>, 300);
export const getExchangeRate = withCache('exchange-rate', fetchExchangeRate as () => Promise<number>, 600);
export const getTransactions = withCache('transactions:all', fetchTransactions as () => Promise<Transaction[]>, 300);
export const getGoals = withCache('goals:all', fetchGoals as () => Promise<any[]>, 300);
export const getLoans = withCache('loans:all', fetchLoans as () => Promise<any[]>, 300);
export { addTransaction };

// Server-only stubs for search/market functions that would import fs
export async function executeSearch(query?: string): Promise<SearchResponse> {
  return { results: [] };
}

export async function getMarketPulseData(period?: string, force?: boolean): Promise<MarketPulseResponse> {
  return {
    us: {
      assets: [],
      drivers: { topMovers: [], summaryEn: '', summaryVi: '' },
      capitalFlow: { signal: 'MIXED', summaryEn: '', summaryVi: '' },
      correlationMatrix: [],
      assetList: [],
    },
    vn: {
      assets: [],
      drivers: { topMovers: [], summaryEn: '', summaryVi: '' },
      capitalFlow: { signal: 'MIXED', summaryEn: '', summaryVi: '' },
      correlationMatrix: [],
      assetList: [],
    },
    lastUpdated: new Date().toISOString(),
  };
}

export async function getAnalyzedNews(topic?: string) {
  return { articles: [], analysis: '' };
}
