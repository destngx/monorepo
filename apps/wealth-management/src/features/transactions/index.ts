/**
 * Transactions Feature - Public API
 * 
 * Only export what's needed by other features and pages.
 * Internal implementation details (api/, model/internal) are private.
 */

// UI Components
export { TransactionsPage, TransactionTable, TransactionForm, TransactionFilters, TransactionReviewAI, NotificationProcessor } from './ui';

// Model/Business Logic
export { useTransactions, useTransactionsByAccount, getTransactions, getTransactionsByAccount } from './model';

// Types
export type { Transaction, TransactionType } from './model';
