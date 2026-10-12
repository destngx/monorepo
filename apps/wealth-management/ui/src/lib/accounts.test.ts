import { describe, expect, it } from 'vitest';
import type { Account } from './api/types';
import { creditRemaining, creditUtilization, groupAccounts, noteAmount } from './accounts';

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
