import type { Account, AccountType } from '$lib/api/types';

export interface AccountForm {
  name: string;
  type: AccountType;
  currency: string;
  balance: number | null;
  note: string;
}

export const emptyAccountForm = (): AccountForm => ({
  name: '',
  type: 'active use',
  currency: 'VND',
  balance: null,
  note: '',
});

/** Form seeded from an existing account; balances are never edited here (WM-ACC-7). */
export const accountFormFor = (account: Account): AccountForm => ({
  name: account.name,
  type: account.type,
  currency: account.currency,
  balance: null,
  note: account.note ?? '',
});
