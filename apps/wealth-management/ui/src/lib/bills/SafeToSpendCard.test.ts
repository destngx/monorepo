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
    spendableCash: 20_000_000,
    cashAccounts: [],
    billsDue: 3_000_000,
    cardPayments: 5_000_000,
    goalSetAside: 4_000_000,
    goalPlanUsed: true,
    amount: safeToSpend - shortBy,
    safeToSpend,
    shortBy,
    perDay: safeToSpend / 8,
    counted: [],
    explanation: '',
    goalSetAsideRule: '',
  },
  facts: {
    items: 3,
    upcoming30Days: 8_000_000,
    upcomingCount: 2,
    overdueCount: 0,
    overdueAmount: 0,
    safeToSpend,
    shortBy,
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

  it('says when the money is short', () => {
    mask.toggle();
    const { container } = render(SafeToSpendCard, { summary: summary(0, 2_000_000) });
    expect(container.querySelector('[data-status="serious"]')?.textContent).toContain('Short');
    expect(container.textContent).toContain('Short by 2.000.000');
  });
});
