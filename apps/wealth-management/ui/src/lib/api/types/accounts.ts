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
  goalAmount: number | null;
  goalProgress: number | null;
  clearedBalance: number;
  balance: number;
  type: AccountType;
  currency: Currency;
  note: string | null;
}
