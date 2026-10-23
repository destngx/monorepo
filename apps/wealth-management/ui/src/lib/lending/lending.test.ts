import { describe, expect, it } from 'vitest';
import type { LentLoan, MoneyLentPerson } from '$lib/api/types';
import { ageTone, balanceStatus, formatAge, otherOpenLoans, repayStatus, splitPeople } from './lending';

const person = (name: string, outstanding: number): MoneyLentPerson => ({
  name,
  lent: Math.max(outstanding, 0),
  repaid: Math.max(-outstanding, 0),
  outstanding,
  lastActivity: '2026-10-01',
  openSince: outstanding === 0 ? null : '2026-09-01',
  daysOutstanding: outstanding === 0 ? null : 30,
  repayBy: null,
  daysUntilRepay: null,
  repayByAmount: 0,
  loans: [],
  transactions: [],
});

const loan = (outstanding: number, repayBy: string | null): LentLoan => ({
  transactionId: 'row-2',
  date: '2026-10-01',
  amount: 1_000_000,
  repaid: 1_000_000 - outstanding,
  outstanding,
  daysOutstanding: 22,
  repayBy,
  daysUntilRepay: repayBy === null ? null : 8,
});

describe('money lent', () => {
  it('keeps open balances apart from settled people, in order', () => {
    const { open, settled } = splitPeople([
      person('Person A', 10_000_000),
      person('Person B', -2_000_000),
      person('Person C', 0),
    ]);
    expect(open.map((p) => p.name)).toEqual(['Person A', 'Person B']);
    expect(settled.map((p) => p.name)).toEqual(['Person C']);
  });

  it('says who owes whom', () => {
    expect(balanceStatus(1)).toEqual({ tone: 'info', label: 'Owes you' });
    expect(balanceStatus(-1)).toEqual({ tone: 'warning', label: 'You owe' });
    expect(balanceStatus(0)).toEqual({ tone: 'good', label: 'Settled' });
  });

  it('writes the age in days, months or years', () => {
    expect(formatAge(null)).toBe('-');
    expect(formatAge(1)).toBe('1 day');
    expect(formatAge(45)).toBe('45 days');
    expect(formatAge(200)).toBe('6 months');
    expect(formatAge(1095)).toBe('3.0 years');
  });

  it('flags old balances', () => {
    expect(ageTone(30)).toBe('neutral');
    expect(ageTone(120)).toBe('warning');
    expect(ageTone(400)).toBe('serious');
    expect(ageTone(null)).toBe('neutral');
  });

  it('flags a repayment date only when it is close or has passed', () => {
    expect(repayStatus(null)).toBeNull();
    expect(repayStatus(10)).toEqual({ tone: 'neutral', label: 'Due in 10 days' });
    expect(repayStatus(3)).toEqual({ tone: 'warning', label: 'Due in 3 days' });
    expect(repayStatus(1)).toEqual({ tone: 'warning', label: 'Due tomorrow' });
    expect(repayStatus(0)).toEqual({ tone: 'warning', label: 'Due today' });
    expect(repayStatus(-1)).toEqual({ tone: 'serious', label: '1 day overdue' });
    expect(repayStatus(-5)).toEqual({ tone: 'serious', label: '5 days overdue' });
  });

  it('counts the open loans the soonest repayment date leaves out', () => {
    const owing = {
      ...person('Person A', 2_000_000),
      repayBy: '2026-10-31',
      loans: [
        loan(0, null),
        loan(500_000, '2026-10-31'),
        loan(500_000, '2026-10-31'),
        loan(1_000_000, '2026-11-30'),
        loan(0, null),
        loan(500_000, null),
      ],
    };
    expect(otherOpenLoans(owing)).toBe(2);
    expect(otherOpenLoans(person('Person B', 1_000_000))).toBe(0);
  });
});
