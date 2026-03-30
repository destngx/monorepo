/**
 * Account Type Configuration
 *
 * Central source of truth for account types with full metadata.
 * Moved from features/accounts/model/types.ts to enable centralized
 * configuration and easier access from app layer.
 */

/**
 * Comprehensive account type enumeration
 * Defines all possible account classifications in the system
 */
export type AccountType =
  | 'active use'
  | 'rarely use'
  | 'long holding'
  | 'deprecated'
  | 'negative active use'
  | 'bank'
  | 'crypto'
  | 'cash'
  | 'investment';

/**
 * Metadata for account type display and categorization
 */
export interface AccountTypeMetadata {
  /** The account type value */
  id: AccountType;
  /** Display label for UI */
  label: string;
  /** Icon identifier (e.g., for icon library) */
  icon: string;
  /** Color code for visual representation */
  color: string;
  /** User-friendly description */
  description: string;
}

/**
 * Complete account type metadata configuration
 * Used for rendering, categorization, and business logic
 */
export const ACCOUNT_TYPES: Record<AccountType, AccountTypeMetadata> = {
  'active use': {
    id: 'active use',
    label: 'Active Use',
    icon: 'wallet',
    color: 'bg-blue-500',
    description: 'Frequently used accounts for daily transactions and spending',
  },
  'rarely use': {
    id: 'rarely use',
    label: 'Rarely Used',
    icon: 'inbox',
    color: 'bg-slate-400',
    description: 'Accounts accessed infrequently, maintained for backup or special purposes',
  },
  'long holding': {
    id: 'long holding',
    label: 'Long Holding',
    icon: 'piggy-bank',
    color: 'bg-green-500',
    description: 'Accounts intended for long-term savings or investment',
  },
  deprecated: {
    id: 'deprecated',
    label: 'Deprecated',
    icon: 'archive',
    color: 'bg-red-500',
    description: 'Inactive or closed accounts maintained for historical reference',
  },
  'negative active use': {
    id: 'negative active use',
    label: 'Negative Active Use',
    icon: 'trending-down',
    color: 'bg-orange-500',
    description: 'Accounts with negative balance (liabilities, debts, loans)',
  },
  bank: {
    id: 'bank',
    label: 'Bank Account',
    icon: 'bank',
    color: 'bg-indigo-500',
    description: 'Traditional bank accounts (checking, savings, money market)',
  },
  crypto: {
    id: 'crypto',
    label: 'Cryptocurrency',
    icon: 'bitcoin',
    color: 'bg-yellow-500',
    description: 'Cryptocurrency wallets and exchange accounts',
  },
  cash: {
    id: 'cash',
    label: 'Cash',
    icon: 'dollar-sign',
    color: 'bg-green-600',
    description: 'Physical cash holdings',
  },
  investment: {
    id: 'investment',
    label: 'Investment Account',
    icon: 'trending-up',
    color: 'bg-purple-500',
    description: 'Brokerage accounts, mutual funds, stocks, bonds',
  },
};
