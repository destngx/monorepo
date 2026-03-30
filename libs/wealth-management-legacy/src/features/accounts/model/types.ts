// Re-export AccountType from config for backward compatibility
export type { AccountType } from '../../../config/accounts/types';
export { ACCOUNT_TYPES, type AccountTypeMetadata } from '../../../config/accounts/types';
export { inferCurrency, CRYPTO_ACCOUNTS, USD_ACCOUNTS } from '../../../config/accounts/rules';

export type Currency = 'VND' | 'USD' | 'USDT';

export interface Account {
  name: string;
  dueDate: string | null;
  goalAmount: number | null;
  goalProgress: number | null;
  clearedBalance: number;
  balance: number;
  type: string;
  currency: Currency;
  note: string | null;
}
