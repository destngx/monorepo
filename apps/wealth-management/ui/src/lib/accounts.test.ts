import { describe, expect, it } from 'vitest';
import type { Account } from './api/types';
import {
  adjustmentHref,
  allocationByGroup,
  creditRemaining,
  creditUtilization,
  fromVND,
  groupAccounts,
  LIQUIDITY_BY_TYPE,
  liquidityBreakdown,
  liquidityTier,
  noteAmount,
  sumInVND,
  toVND,
  totalHistory,
  unconvertible,
} from './accounts';
import { ACCOUNT_TYPES } from './api/types';

const account = (overrides: Partial<Account>): Account => ({
  name: 'A',
  dueDate: null,
  goalAmount: null,
  goalProgress: null,
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
    expect(creditUtilization(account({ balance: -45, goalAmount: 50 }))).toBe(90);
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

const rates = { USD: 25_000, USDT: 26_000 };

describe('currency conversion (WM-ACC-8)', () => {
  it('converts to VND by currency and back to the display currency', () => {
    expect(toVND(10, 'USD', rates)).toBe(250_000);
    expect(toVND(10, 'VND', {})).toBe(10);
    expect(toVND(10, 'USDT', {})).toBeNull();
    expect(fromVND(250_000, 'USD', rates)).toBe(10);
    expect(fromVND(250_000, 'VND', {})).toBe(250_000);
    expect(fromVND(1, 'USD', {})).toBeNull();
  });

  it('sums mixed-currency balances in VND', () => {
    expect(sumInVND([account({ balance: 100 }), account({ currency: 'USD', balance: 2 })], { USD: 50 })).toBe(200);
  });

  it('lists accounts that have no rate', () => {
    const missing = unconvertible([account({ name: 'A' }), account({ name: 'B', currency: 'USDT', balance: 1 })], {
      USD: 25_000,
    });
    expect(missing.map((a) => a.name)).toEqual(['B']);
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

  it('sums positive converted balances per tier with shares', () => {
    const tiers = liquidityBreakdown(
      [
        account({ type: 'active use', balance: 500_000 }),
        account({ type: 'crypto', currency: 'USD', balance: 10 }),
        account({ type: 'long holding', balance: 250_000 }),
        account({ type: 'negative active use', balance: -900_000 }),
        account({ type: 'active use', balance: -100 }),
      ],
      rates,
    );
    expect(tiers).toEqual([
      { tier: 'Immediate', value: 500_000, percent: 50 },
      { tier: 'T+2', value: 250_000, percent: 25 },
      { tier: 'Illiquid', value: 250_000, percent: 25 },
    ]);
  });
});

describe('allocationByGroup', () => {
  it('totals groups in VND and leaves out credit, archived and non-positive groups', () => {
    const slices = allocationByGroup(
      [
        account({ type: 'active use', balance: 100 }),
        account({ type: 'cash', balance: 50 }),
        account({ type: 'crypto', currency: 'USD', balance: 2 }),
        account({ type: 'negative active use', balance: -500 }),
        account({ type: 'deprecated', balance: 30 }),
        account({ type: 'rarely use', balance: 0 }),
      ],
      { USD: 100 },
    );
    expect(slices).toEqual([
      { label: 'Everyday Banking', value: 150 },
      { label: 'Investments', value: 200 },
    ]);
  });
});

describe('totalHistory', () => {
  it('sums converted daily balances by date', () => {
    const points = totalHistory(
      {
        Bank: [
          { date: '2026-10-02', value: 100 },
          { date: '2026-10-01', value: 90 },
        ],
        Usd: [{ date: '2026-10-01', value: 1 }],
        Coin: [{ date: '2026-10-01', value: 5 }],
      },
      [
        account({ name: 'Bank' }),
        account({ name: 'Usd', currency: 'USD' }),
        account({ name: 'Coin', currency: 'USDT' }),
      ],
      { USD: 10 },
    );
    expect(points).toEqual([
      { date: '2026-10-01', value: 100 },
      { date: '2026-10-02', value: 100 },
    ]);
  });
});

describe('adjustmentHref (WM-ACC-7)', () => {
  it('pre-fills an adjustment transaction for the account', () => {
    expect(adjustmentHref('VCB & Co')).toBe('/transactions?new=1&account=VCB%20%26%20Co&payee=Balance%20adjustment');
  });
});
