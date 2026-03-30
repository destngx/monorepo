import type { AccountType } from './types';

export type Currency = 'VND' | 'USD' | 'USDT';

export const CRYPTO_ACCOUNTS = ['binance'];
export const USD_ACCOUNTS: string[] = [];

export function inferCurrency(name: string, type: string): Currency {
  const lower = name.toLowerCase();
  if (CRYPTO_ACCOUNTS.some((c) => lower.includes(c))) return 'USDT';
  if (USD_ACCOUNTS.some((c) => lower.includes(c))) return 'USD';
  return 'VND';
}

export interface AccountRules {
  cryptoAccounts: string[];
  usdAccounts: string[];
}

export const ACCOUNT_RULES: AccountRules = {
  cryptoAccounts: CRYPTO_ACCOUNTS,
  usdAccounts: USD_ACCOUNTS,
};

export function classifyAccountType(type: AccountType): 'asset' | 'liability' | 'neutral' {
  switch (type) {
    case 'negative active use':
    case 'deprecated':
      return 'liability';
    case 'active use':
    case 'rarely use':
    case 'long holding':
    case 'bank':
    case 'crypto':
    case 'cash':
    case 'investment':
      return 'asset';
    default:
      return 'neutral';
  }
}

export function isAccountActive(type: AccountType): boolean {
  return type !== 'deprecated';
}
