import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { BillsSummary } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import SafeToSpendCard from './SafeToSpendCard.svelte';

vi.mock('$app/paths', () => ({ base: '/ui' }));

const summary = (safeToSpend: number, shortBy = 0): BillsSummary => ({
  safeToSpend: {
    asOf: '2026-10-17',
    until: '2026-10-25',
    untilIncome: true,
    incomeName: 'Salary',
    days: 8,
    readyMoney: 20_000_000,
    readyAccounts: [{ name: 'Bank A', balance: 25_000_000, reserved: 5_000_000, spendable: 20_000_000 }],
    excludedAccounts: [{ name: 'Broker B', balance: 9_000_000, reason: 'not_ready' }],
    readiness: { notReadyAccounts: ['Broker B'], etag: 'etag-1' },
    reserved: 5_000_000,
    minimumBalance: { accounts: [{ account: 'Bank A', minimum: 5_000_000 }], etag: 'etag-m' },
    minimumBalances: [],
    minimumRule: '',
    unknownMinimum: [],
    billsDue: 3_000_000,
    cardPayments: 5_000_000,
    amount: safeToSpend - shortBy,
    safeToSpend,
    shortBy,
    perDay: safeToSpend / 8,
    counted: [],
    explanation: '',
    readyRule: '',
    unknownNotReady: [],
    issues: [],
  },
  facts: {
    items: 3,
    upcoming30Days: 8_000_000,
    upcomingCount: 2,
    overdueCount: 0,
    overdueAmount: 0,
    safeToSpend,
    shortBy,
    readyMoney: 20_000_000,
    dueBeforeUntil: 8_000_000,
    excludedAccounts: 1,
    minimumReserved: 5_000_000,
    minimumBalanceAccounts: 1,
    minimumBalanceAtRisk: 0,
    minimumBalanceExtraNeeded: 0,
    until: '2026-10-25',
    untilIncome: true,
    suggestions: 0,
  },
  overdue: [],
  dueSoon: [],
});

describe('SafeToSpendCard', () => {
  afterEach(() => mask.hide());

  it('shows what is left until the next income with a status badge, masked by default', () => {
    const { container } = render(SafeToSpendCard, { summary: summary(8_000_000) });
    expect(screen.getByTestId('safe-amount').textContent).toBe(MASK);
    expect(container.querySelector('[data-status="good"]')?.textContent).toContain('Covered');
    expect(container.textContent).toContain('a day for 8 days');
    expect(container.querySelector('a')?.getAttribute('href')).toBe('/ui/bills');
  });

  it('Given ready money and bills before payday When shown Then the math names ready money and no goal set-aside', () => {
    mask.toggle();
    render(SafeToSpendCard, { summary: summary(12_000_000) });
    const math = (screen.getByTestId('safe-math').textContent ?? '').replace(/\s+/g, ' ');
    expect(math).toContain('20.000.000');
    expect(math).toContain('of ready money after');
    expect(math).toContain('8.000.000');
    expect(math).toContain('due before');
    expect(math).toContain('paid by that wage');
    expect(math).toContain('Left out: 1 account');
    expect(math).toContain('5.000.000 ₫ is kept as minimum balances');
    expect(math.toLowerCase()).not.toContain('goal');
  });

  it('says when the money is short', () => {
    mask.toggle();
    const { container } = render(SafeToSpendCard, { summary: summary(0, 2_000_000) });
    expect(container.querySelector('[data-status="serious"]')?.textContent).toContain('Short');
    expect(container.textContent).toContain('Short by 2.000.000');
  });
});
