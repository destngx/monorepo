/**
 * Accounts Feature - Public API
 * 
 * Only export what's needed by other features and pages.
 * Internal implementation details (api/, model/internal) are private.
 */

// UI Components
export { AccountsPage, AccountReviewAI, AccountTrendSparkline, CreditCardSummaryAI, EfficiencyChart } from './ui';

// Model/Business Logic
export { useAccounts, useAccountById, getAccounts, getAccountById } from './model';

// Types
export type { Account, AccountType, Currency } from './model';
