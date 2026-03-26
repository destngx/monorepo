import { withCache } from '../utils/cache';
import { getAccounts as fetchAccounts } from './sheets/accounts';
import { getBudget as fetchBudget } from './sheets/budget';
import { getCategories as fetchCategories, Category } from './sheets/categories';
import { getExchangeRate as fetchExchangeRate } from './services/exchange-rate-service';
import { getTransactions as fetchTransactions, addTransaction } from './sheets/transactions';
import { Account } from '../types/accounts';
import { BudgetItem } from '../types/budget';
import { Transaction } from '../types/transactions';

export * from './index';

// Google Sheets integration (server-only)
export * from './sheets/auth';
export * from './sheets/client';
export * from './sheets/goals';
export * from './sheets/loans';
export * from './sheets/notifications';
export * from './sheets/mappers';

// Cached service exports
export const getAccounts = withCache('accounts:all', fetchAccounts as () => Promise<Account[]>, 300);
export const getBudget = withCache('budget:all', fetchBudget as () => Promise<BudgetItem[]>, 300);
export const getCategories = withCache('categories:all', fetchCategories as () => Promise<Category[]>, 300);
export const getExchangeRate = withCache('exchange-rate', fetchExchangeRate as () => Promise<number>, 600);
export const getTransactions = withCache('transactions:all', fetchTransactions as () => Promise<Transaction[]>, 300);
export { addTransaction };
