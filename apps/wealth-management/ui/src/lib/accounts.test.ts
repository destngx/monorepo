import { describe, expect, it } from 'vitest';
import type { Account } from './api/types';
import {
  adjustmentHref,
  creditLines,
  creditStandings,
  groupAccounts,
  LIQUIDITY_BY_TYPE,
  liquidityBreakdown,
  liquidityTier,
  noteAmount,
  sumBalances,
  totalHistory,
  walletProducts,
} from './accounts';
import { ACCOUNT_TYPES, type CardBankReport, type FlexibleSaving } from './api/types';

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
  it('reads a number from a note', () => {
    expect(noteAmount('Còn 12.500.000')).toBe(12500000);
    expect(noteAmount('3,000,000')).toBe(3000000);
    expect(noteAmount('due soon')).toBeNull();
  });
});

describe('credit standings (WM-ACC-3)', () => {
  const card = (name: string, balance: number, creditLimit: number | null = null) =>
    account({ name, balance, creditLimit, type: 'negative active use' });
  const accounts = [
    card('Card Acme', -300_000),
    card('Card Acme Extra', -200_000),
    card('Card Solo', -1_500_000, 2_000_000),
    card('Card Ahead', 100_000, 1_000_000),
    card('Card Nolimit', -50_000),
    account({ name: 'Bank', balance: 1_000_000 }),
  ];
  const reports = [
    {
      limit: 5_000_000,
      cards: [
        { accountName: 'Card Acme', keyword: 'platinum' },
        { accountName: 'Card Acme Extra', keyword: 'gold' },
      ],
    },
  ] as unknown as CardBankReport[];
  const standings = creditStandings(accounts, creditLines(reports, accounts));

  it('takes a shared bank limit less what every account on it owes', () => {
    expect(standings.get('Card Acme')).toEqual({
      limit: 5_000_000,
      owed: 500_000,
      available: 4_500_000,
      utilization: 10,
      sharedWith: ['Card Acme Extra'],
    });
    expect(standings.get('Card Acme Extra')?.sharedWith).toEqual(['Card Acme']);
    expect(standings.get('Card Acme Extra')?.available).toBe(4_500_000);
  });

  it("uses the account's own limit when no bank sets one", () => {
    expect(standings.get('Card Solo')).toEqual({
      limit: 2_000_000,
      owed: 1_500_000,
      available: 500_000,
      utilization: 75,
      sharedWith: [],
    });
  });

  it('does not count money paid in ahead as extra room', () => {
    expect(standings.get('Card Ahead')).toMatchObject({ owed: 0, available: 1_000_000, utilization: 0 });
  });

  it('gives no standing to a card without a limit or to other accounts', () => {
    expect(standings.has('Card Nolimit')).toBe(false);
    expect(standings.has('Bank')).toBe(false);
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

describe('walletProducts (D41)', () => {
  // Invented products and rates.
  const product = (id: string, rate: number): FlexibleSaving => ({
    id,
    provider: id,
    product: 'Earning balance',
    rate,
    backing: 'deposit',
    earningStarts: 'next_working_day',
    url: 'https://example.test',
  });
  const products = [product('wallet-a', 4.1), product('wallet-b', 3.6)];

  it('Given accounts linked to wallet products When joined Then each linked account gets its product, by name case-insensitively', () => {
    const linked = walletProducts(
      [account({ name: 'Wallet A' }), account({ name: 'Wallet B' }), account({ name: 'Bank' })],
      [
        { account: ' wallet a ', product: 'wallet-a' },
        { account: 'Wallet B', product: 'wallet-b' },
      ],
      products,
    );
    expect([...linked.entries()].map(([name, p]) => [name, p.rate])).toEqual([
      ['Wallet A', 4.1],
      ['Wallet B', 3.6],
    ]);
  });

  it('Given a link to a product the server does not list today When joined Then the account gets no rate', () => {
    const linked = walletProducts(
      [account({ name: 'Wallet A' })],
      [{ account: 'Wallet A', product: 'gone' }],
      products,
    );
    expect(linked.size).toBe(0);
  });
});
