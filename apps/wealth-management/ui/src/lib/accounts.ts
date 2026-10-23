import type { Account, AccountType, CardBankReport, FlexibleSaving, TrendPoint } from './api/types';
import { type IconName } from '@libs/ui-kit';

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

/** A credit limit and the card accounts that draw on it. */
export interface CreditLine {
  limit: number;
  accounts: string[];
}

/**
 * Each card account's credit line: its bank's limit, shared by every card
 * account the bank's cards bill to (WM-CARD-16), else the account's own
 * limit (Accounts C). Accounts with neither have no line.
 */
export function creditLines(reports: CardBankReport[], accounts: Account[]): Map<string, CreditLine> {
  const cards = new Map(accounts.filter(isCreditAccount).map((account) => [account.name, account]));
  const lines = new Map<string, CreditLine>();
  for (const bank of reports) {
    const names = [...new Set(bank.cards.map((card) => card.accountName))].filter((name) => cards.has(name));
    if (bank.limit <= 0 || !names.length) continue;
    const line = { limit: bank.limit, accounts: names };
    for (const name of names) lines.set(name, line);
  }
  for (const account of cards.values()) {
    if (!lines.has(account.name) && (account.creditLimit ?? 0) > 0) {
      lines.set(account.name, { limit: account.creditLimit!, accounts: [account.name] });
    }
  }
  return lines;
}

/** What the line's accounts owe; money paid in ahead is not extra room, as on the Cards page. */
function owedOn(line: CreditLine, balances: Map<string, number>): number {
  return line.accounts.reduce((sum, name) => sum + Math.max(0, -(balances.get(name) ?? 0)), 0);
}

/** The limit less what the line's accounts owe. */
export function creditLeft(line: CreditLine, balances: Map<string, number>): number {
  return line.limit - owedOn(line, balances);
}

/** A card account's place on its credit line (WM-ACC-3). */
export interface CreditStanding {
  limit: number;
  /** What every account on the line owes. */
  owed: number;
  available: number;
  /** owed / limit, in percent. */
  utilization: number;
  /** The other card accounts drawing on the same limit. */
  sharedWith: string[];
}

/** Each card account with a credit line, by name, from the accounts' balances. */
export function creditStandings(accounts: Account[], lines: Map<string, CreditLine>): Map<string, CreditStanding> {
  const balances = new Map(accounts.map((account) => [account.name, account.balance]));
  const standings = new Map<string, CreditStanding>();
  for (const [name, line] of lines) {
    const owed = owedOn(line, balances);
    standings.set(name, {
      limit: line.limit,
      owed,
      available: line.limit - owed,
      utilization: (owed / line.limit) * 100,
      sharedWith: line.accounts.filter((other) => other !== name),
    });
  }
  return standings;
}

/**
 * The wallet product each linked account earns (D41), by account name. Links
 * match names case-insensitively, as the idle cash settings do; a link to a
 * product the market data server does not list today gives no rate.
 */
export function walletProducts(
  accounts: Account[],
  links: { account: string; product: string }[],
  products: FlexibleSaving[],
): Map<string, FlexibleSaving> {
  const key = (name: string) => name.trim().toLowerCase();
  const byAccount = new Map(links.map((link) => [key(link.account), link.product]));
  const linked = new Map<string, FlexibleSaving>();
  for (const account of accounts) {
    const product = products.find((p) => p.id === byAccount.get(key(account.name)));
    if (product) linked.set(account.name, product);
  }
  return linked;
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

// Balances are VND (D25): the sheet converts USD/USDT entries with a formula
// when they are written, so account.currency is only the entry currency.

/** Sum of balances in VND. */
export function sumBalances(accounts: Account[]): number {
  return accounts.reduce((sum, account) => sum + account.balance, 0);
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
export function liquidityBreakdown(accounts: Account[]): { tier: LiquidityTier; value: number; percent: number }[] {
  const sums = new Map<LiquidityTier, number>(LIQUIDITY_TIERS.map((tier) => [tier, 0]));
  for (const account of accounts) {
    const tier = liquidityTier(account.type);
    if (!tier || account.balance <= 0) continue;
    sums.set(tier, (sums.get(tier) ?? 0) + account.balance);
  }
  const total = [...sums.values()].reduce((sum, value) => sum + value, 0);
  return LIQUIDITY_TIERS.map((tier) => {
    const value = sums.get(tier) ?? 0;
    return { tier, value, percent: total ? (value / total) * 100 : 0 };
  });
}

/** Total balance history in VND: the accounts' daily series summed by date. */
export function totalHistory(histories: Record<string, TrendPoint[]>, accounts: Account[]): TrendPoint[] {
  const byDate = new Map<string, number>();
  for (const account of accounts) {
    for (const point of histories[account.name] ?? []) {
      byDate.set(point.date, (byDate.get(point.date) ?? 0) + point.value);
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
