import { render, screen, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { defer, settled, type Streamed, type Settled } from '$lib/api/load';
import type { CashFlow, DashboardSummary } from '$lib/api/types';

// Every engine call the cards make on their own (the AI briefing) stays pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import HomePage from './+page.svelte';

const flow = (month: string, income: number, expense: number): CashFlow => ({
  month,
  income,
  expense,
  cashFlow: income - expense,
  savingsRate: income ? (income - expense) / income : 0,
});

// Invented figures.
const summary: DashboardSummary = {
  asOf: '2026-10-15',
  balanceSheet: { assets: 900, liabilities: 100, accountsNet: 800, receivables: 0, payables: 0, netWorth: 800 },
  netGrowthThisMonth: 10,
  netWorthChangePercent: 1.5,
  netWorthTrend: [
    { date: '2026-09-15', value: 780 },
    { date: '2026-10-15', value: 800 },
  ],
  thisMonth: flow('2026-10', 100, 60),
  lastMonth: flow('2026-09', 120, 70),
  lastMonthToDate: flow('2026-09', 90, 50),
  comparedDays: 15,
  incomeChangePercent: 11,
  expenseChangePercent: 20,
  expensePace: null,
  cashFlowHistory: [flow('2026-09', 120, 70), flow('2026-10', 100, 60)],
  topAccounts: [],
};

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Home page', () => {
  it('Given no data has arrived When the page renders Then the shell shows with a skeleton per card', () => {
    render(HomePage, {
      data: { engine: never(), summary: never(), usd: never(), needsYou: never(), safeToSpend: never() } as never,
    });

    expect(screen.getByRole('heading', { level: 1, name: 'Dashboard' })).toBeTruthy();
    expect(screen.getByRole('link', { name: /Add transaction/ }).getAttribute('href')).toBe('/ui/transactions?new=1');
    expect(screen.getByRole('button', { name: /Sync now/ })).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Income vs expense' })).toBeTruthy();
    const loading = screen.getAllByRole('status').map((status) => status.textContent?.trim());
    expect(loading).toEqual(
      expect.arrayContaining([
        'Loading what needs you…',
        'Loading safe to spend…',
        'Loading net worth…',
        'Loading savings this month…',
        'Loading income and expense…',
      ]),
    );
  });

  it('Given the summary arrives and safe-to-spend fails When they settle Then the cards fill in and the failure stays in its card', async () => {
    render(HomePage, {
      data: {
        engine: never(),
        summary: settled(summary),
        usd: never(),
        needsYou: never(),
        safeToSpend: defer(Promise.reject(new Error('bills tab missing'))),
      } as never,
    });
    await flush();

    expect(screen.getByText('As of 15/10/2026')).toBeTruthy();
    expect(screen.getByRole('heading', { name: 'Net worth' })).toBeTruthy();
    expect(screen.getByText('Loading the USD value…')).toBeTruthy();
    const failure = screen.getByRole('alert');
    expect(failure.textContent).toContain('Could not load safe to spend');
    expect(failure.textContent).toContain('bills tab missing');
    expect(within(failure).getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('Loading what needs you…')).toBeTruthy();
  });
});
