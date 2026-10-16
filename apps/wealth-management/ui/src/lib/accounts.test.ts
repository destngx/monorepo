import { describe, expect, it } from 'vitest';
import type { Account } from './api/types';
import {
  adjustmentHref,
  creditRemaining,
  creditUtilization,
  groupAccounts,
  LIQUIDITY_BY_TYPE,
  liquidityBreakdown,
  liquidityTier,
  noteAmount,
  sumBalances,
  totalHistory,
} from './accounts';
import { ACCOUNT_TYPES } from './api/types';

const account = (overrides: Partial<Account>): Account => ({
  name: 'A',
  dueDate: null,
  creditLimit: null,
  clearedBalance: 0,
  balance: 0,
  type: 'active use',
  currency: 'VND',
  note: null,
  ...overrides,
});

describe('groupAccounts', () => {
  it('orders groups and keeps every account type visible', () => {
    const groups = groupAccounts([
      account({ name: 'Old', type: 'deprecated', balance: 5 }),
      account({ name: 'Wallet', type: 'cash', balance: 10 }),
      account({ name: 'Visa', type: 'negative active use', balance: -4 }),
      account({ name: 'Bank', type: 'active use', balance: 20 }),
    ]);

    expect(groups.map((g) => g.group.label)).toEqual(['Everyday Banking', 'Credit Cards & Loans', 'Archived']);
    expect(groups[0].total).toBe(30);
    expect(groups[2].group.subtotal).toBe(false);
  });
});

describe('credit helpers', () => {
  it('reads remaining credit from a numeric note, else the balance', () => {
    expect(noteAmount('Còn 12.500.000')).toBe(12500000);
    expect(creditRemaining(account({ note: '3,000,000', balance: -9 }))).toBe(3000000);
    expect(creditRemaining(account({ note: 'due soon', balance: -9 }))).toBe(9);
  });

  it('computes utilisation against the goal amount', () => {
    expect(creditUtilization(account({ balance: -45, creditLimit: 50 }))).toBe(90);
    expect(creditUtilization(account({ balance: -45 }))).toBeNull();
  });
});

describe('Other group (WM-ACC-1)', () => {
  it('collects account types the UI does not know instead of hiding them', () => {
    const groups = groupAccounts([
      account({ name: 'Mystery', type: 'pension' as Account['type'], balance: 7 }),
      account({ name: 'Bank', type: 'active use', balance: 1 }),
    ]);
    expect(groups.map((g) => g.group.label)).toEqual(['Everyday Banking', 'Other']);
    expect(groups[1].accounts.map((a) => a.name)).toEqual(['Mystery']);
    expect(groups[1].total).toBe(7);
  });
});

describe('sumBalances (D25)', () => {
  it('adds balances as VND: the sheet already converted USD/USDT entries', () => {
    expect(sumBalances([account({ balance: 100 }), account({ currency: 'USDT', balance: 2_600_000 })])).toBe(2_600_100);
  });
});

describe('liquidity tiers (WM-ACC-8)', () => {
  it('maps every known account type explicitly', () => {
    expect(Object.keys(LIQUIDITY_BY_TYPE).sort()).toEqual([...ACCOUNT_TYPES].sort());
    expect(liquidityTier('active use')).toBe('Immediate');
    expect(liquidityTier('cash')).toBe('Immediate');
    expect(liquidityTier('rarely use')).toBe('T+2');
    expect(liquidityTier('investment')).toBe('T+2');
    expect(liquidityTier('long holding')).toBe('Illiquid');
    expect(liquidityTier('negative active use')).toBeNull();
    expect(liquidityTier('deprecated')).toBeNull();
    expect(liquidityTier('pension')).toBe('Illiquid');
  });

  it('sums positive balances per tier with shares', () => {
    const tiers = liquidityBreakdown([
      account({ type: 'active use', balance: 500_000 }),
      account({ type: 'crypto', currency: 'USDT', balance: 250_000 }),
      account({ type: 'long holding', balance: 250_000 }),
      account({ type: 'negative active use', balance: -900_000 }),
      account({ type: 'active use', balance: -100 }),
    ]);
    expect(tiers).toEqual([
      { tier: 'Immediate', value: 500_000, percent: 50 },
      { tier: 'T+2', value: 250_000, percent: 25 },
      { tier: 'Illiquid', value: 250_000, percent: 25 },
    ]);
  });
});

describe('totalHistory', () => {
  it('sums daily balances by date, USDT-entry accounts included as VND', () => {
    const points = totalHistory(
      {
        Bank: [
          { date: '2026-10-02', value: 100 },
          { date: '2026-10-01', value: 90 },
        ],
        Coin: [{ date: '2026-10-01', value: 5 }],
      },
      [account({ name: 'Bank' }), account({ name: 'Coin', currency: 'USDT' })],
    );
    expect(points).toEqual([
      { date: '2026-10-01', value: 95 },
      { date: '2026-10-02', value: 100 },
    ]);
  });
});

describe('adjustmentHref (WM-ACC-7)', () => {
  it('pre-fills an adjustment transaction for the account', () => {
    expect(adjustmentHref('VCB & Co')).toBe('/transactions?new=1&account=VCB%20%26%20Co&payee=Balance%20adjustment');
  });
});
