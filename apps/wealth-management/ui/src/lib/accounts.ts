import type { Account, AccountType, TrendPoint } from './api/types';
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

/** Catch-all for types the UI does not know yet, so no account ever disappears (WM-ACC-1). */
export const OTHER_GROUP: AccountGroup = { key: 'other', label: 'Other', types: [], subtotal: true, expanded: true };

/** The group an account type belongs to; unknown types land in Other. */
export function groupFor(type: string): AccountGroup {
  return ACCOUNT_GROUPS.find((group) => (group.types as string[]).includes(type)) ?? OTHER_GROUP;
}

export function groupAccounts(accounts: Account[]): { group: AccountGroup; accounts: Account[]; total: number }[] {
  return [...ACCOUNT_GROUPS, OTHER_GROUP]
    .map((group) => {
      const members = accounts.filter((account) => groupFor(account.type).key === group.key);
      return { group, accounts: members, total: members.reduce((sum, account) => sum + account.balance, 0) };
    })
    .filter((entry) => entry.accounts.length > 0);
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

// Cross-currency view (WM-ACC-8). Rates are VND per unit of the currency.
export type DisplayCurrency = 'VND' | 'USD';
export type VndRates = Partial<Record<string, number>>;

/** Amount converted to VND; null when the currency has no usable rate. */
export function toVND(amount: number, currency: string, rates: VndRates): number | null {
  if (currency === 'VND') return amount;
  const rate = rates[currency];
  return rate && Number.isFinite(rate) && rate > 0 ? amount * rate : null;
}

/** VND amount in the display currency; null when USD is asked for without a rate. */
export function fromVND(vnd: number, display: DisplayCurrency, rates: VndRates): number | null {
  if (display === 'VND') return vnd;
  const rate = rates.USD;
  return rate && Number.isFinite(rate) && rate > 0 ? vnd / rate : null;
}

/** Sum of balances in VND; accounts without a rate are left out (see unconvertible). */
export function sumInVND(accounts: Account[], rates: VndRates): number {
  return accounts.reduce((sum, account) => sum + (toVND(account.balance, account.currency, rates) ?? 0), 0);
}

/** Accounts whose currency cannot be converted, so totals can say what they leave out. */
export function unconvertible(accounts: Account[], rates: VndRates): Account[] {
  return accounts.filter((account) => toVND(account.balance, account.currency, rates) === null);
}

// Liquidity tiers (WM-ACC-8): how fast an account's money can be spent.
export type LiquidityTier = 'Immediate' | 'T+2' | 'Illiquid';
export const LIQUIDITY_TIERS: LiquidityTier[] = ['Immediate', 'T+2', 'Illiquid'];

/**
 * Explicit type-to-tier mapping. Credit accounts are liabilities and archived
 * accounts are not in use, so neither has a tier.
 */
export const LIQUIDITY_BY_TYPE: Record<AccountType, LiquidityTier | null> = {
  'active use': 'Immediate',
  bank: 'Immediate',
  cash: 'Immediate',
  'rarely use': 'T+2',
  investment: 'T+2',
  crypto: 'T+2',
  'long holding': 'Illiquid',
  'negative active use': null,
  deprecated: null,
};

/** Tier for a type; types the UI does not know are treated as Illiquid (the conservative reading). */
export function liquidityTier(type: string): LiquidityTier | null {
  return type in LIQUIDITY_BY_TYPE ? LIQUIDITY_BY_TYPE[type as AccountType] : 'Illiquid';
}

/** Positive balances per liquidity tier in VND, with each tier's share of the tiered total. */
export function liquidityBreakdown(
  accounts: Account[],
  rates: VndRates,
): { tier: LiquidityTier; value: number; percent: number }[] {
  const sums = new Map<LiquidityTier, number>(LIQUIDITY_TIERS.map((tier) => [tier, 0]));
  for (const account of accounts) {
    const tier = liquidityTier(account.type);
    const vnd = toVND(account.balance, account.currency, rates);
    if (!tier || vnd === null || vnd <= 0) continue;
    sums.set(tier, (sums.get(tier) ?? 0) + vnd);
  }
  const total = [...sums.values()].reduce((sum, value) => sum + value, 0);
  return LIQUIDITY_TIERS.map((tier) => {
    const value = sums.get(tier) ?? 0;
    return { tier, value, percent: total ? (value / total) * 100 : 0 };
  });
}

/** Asset allocation by account group in VND: positive group totals only, archived and credit left out. */
export function allocationByGroup(accounts: Account[], rates: VndRates): { label: string; value: number }[] {
  const sums = new Map<string, { label: string; value: number }>();
  for (const account of accounts) {
    const group = groupFor(account.type);
    if (group.key === 'archived' || group.key === 'credit') continue;
    const vnd = toVND(account.balance, account.currency, rates);
    if (vnd === null) continue;
    const entry = sums.get(group.key) ?? { label: group.label, value: 0 };
    entry.value += vnd;
    sums.set(group.key, entry);
  }
  return [...sums.values()].filter((entry) => entry.value > 0);
}

/**
 * Total balance history in VND: each account's daily series converted at
 * today's rate and summed by date. Accounts without a rate are skipped
 * (callers list them via unconvertible).
 */
export function totalHistory(
  histories: Record<string, TrendPoint[]>,
  accounts: Account[],
  rates: VndRates,
): TrendPoint[] {
  const byDate = new Map<string, number>();
  for (const account of accounts) {
    for (const point of histories[account.name] ?? []) {
      const vnd = toVND(point.value, account.currency, rates);
      if (vnd === null) continue;
      byDate.set(point.date, (byDate.get(point.date) ?? 0) + vnd);
    }
  }
  return [...byDate.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([date, value]) => ({ date, value }));
}

/**
 * Link (relative to the app base) to the add-transaction form pre-filled with
 * a balance adjustment for the account (WM-ACC-7: balances change only
 * through transactions).
 */
export function adjustmentHref(accountName: string): string {
  const query = new URLSearchParams({ new: '1', account: accountName, payee: 'Balance adjustment' });
  return `/transactions?${query.toString().replace(/\+/g, '%20')}`;
}
