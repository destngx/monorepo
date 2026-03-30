export type { AccountType } from '@wealth-management/features/accounts/model/types';
export { inferCurrency } from '@wealth-management/features/accounts/model/types';

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
