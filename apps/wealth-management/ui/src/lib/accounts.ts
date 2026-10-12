import type { Account, AccountType } from './api/types';
import type { IconName } from './components/Icon.svelte';

export interface AccountGroup {
  key: string;
  label: string;
  types: AccountType[];
  /** Archived accounts get no subtotal (WM-ACC-1). */
  subtotal: boolean;
  expanded: boolean;
}

// Group order and labels from WM-ACC-1; every type renders somewhere.
export const ACCOUNT_GROUPS: AccountGroup[] = [
  { key: 'everyday', label: 'Everyday Banking', types: ['active use', 'bank', 'cash'], subtotal: true, expanded: true },
  {
    key: 'investments',
    label: 'Investments',
    types: ['long holding', 'investment', 'crypto'],
    subtotal: true,
    expanded: false,
  },
  { key: 'savings', label: 'Savings & Reserves', types: ['rarely use'], subtotal: true, expanded: false },
  { key: 'credit', label: 'Credit Cards & Loans', types: ['negative active use'], subtotal: true, expanded: true },
  { key: 'archived', label: 'Archived', types: ['deprecated'], subtotal: false, expanded: false },
];

export function groupAccounts(accounts: Account[]): { group: AccountGroup; accounts: Account[]; total: number }[] {
  return ACCOUNT_GROUPS.map((group) => {
    const members = accounts.filter((account) => group.types.includes(account.type));
    return { group, accounts: members, total: members.reduce((sum, account) => sum + account.balance, 0) };
  }).filter((entry) => entry.accounts.length > 0);
}

export const isCreditAccount = (account: Account) => account.type === 'negative active use';

/** Number in a note such as "50000000" or "Còn 12.500.000"; null if none. */
export function noteAmount(note: string | null): number | null {
  const match = /-?\d[\d.,]*/.exec(note ?? '');
  if (!match) return null;
  const value = Number(match[0].replace(/[.,]/g, ''));
  return Number.isFinite(value) ? value : null;
}

/** Credit remaining: the numeric note if present, else |balance| (WM-DASH-3). */
export function creditRemaining(account: Account): number {
  return noteAmount(account.note) ?? Math.abs(account.balance);
}

/** Credit utilisation in percent from |balance| / goalAmount (WM-ACC-3). */
export function creditUtilization(account: Account): number | null {
  if (!account.goalAmount) return null;
  return (Math.abs(account.balance) / account.goalAmount) * 100;
}

export function accountIcon(type: AccountType): IconName {
  switch (type) {
    case 'negative active use':
      return 'card';
    case 'long holding':
    case 'investment':
    case 'crypto':
      return 'investments';
    case 'rarely use':
      return 'goal';
    default:
      return 'accounts';
  }
}
