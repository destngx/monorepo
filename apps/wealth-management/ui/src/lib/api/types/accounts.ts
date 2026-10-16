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

export const ACCOUNT_TYPES: AccountType[] = [
  'active use',
  'rarely use',
  'long holding',
  'negative active use',
  'deprecated',
  'bank',
  'crypto',
  'cash',
  'investment',
];

export type Currency = 'VND' | 'USD' | 'USDT';
export type CategoryType = 'income' | 'expense' | 'non-budget';

export interface Account {
  name: string;
  dueDate: string | null;
  /** Accounts column C: a card account's credit limit; null when blank. */
  creditLimit: number | null;
  clearedBalance: number;
  balance: number;
  type: AccountType;
  currency: Currency;
  note: string | null;
}
