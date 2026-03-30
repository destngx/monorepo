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
export type Currency = 'VND' | 'USD' | 'USDT';

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

// Known crypto/USD accounts — used to infer currency when the sheet doesn't have a currency column
const CRYPTO_ACCOUNTS = ['binance'];
const USD_ACCOUNTS: string[] = [];

export function inferCurrency(name: string, type: string): Currency {
  const lower = name.toLowerCase();
  if (CRYPTO_ACCOUNTS.some((c) => lower.includes(c))) return 'USDT';
  if (USD_ACCOUNTS.some((c) => lower.includes(c))) return 'USD';
  return 'VND';
}
