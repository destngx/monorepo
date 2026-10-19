import { render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { settled, type Streamed, type Settled } from '$lib/api/load';
import type { BudgetLine, BudgetReport } from '$lib/api/types';

// Sections that load on their own (recommendations, advisor, the wall) stay pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn(), invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$app/state', () => ({ page: { url: new URL('http://127.0.0.1:5173/ui/budget?month=2026-09') } }));

import BudgetPage from './+page.svelte';

const line = (category: string, spent: number, limit: number): BudgetLine => ({
  category,
  categoryType: 'expense',
  limit,
  spent,
  remaining: limit - spent,
  percentUsed: limit ? (spent / limit) * 100 : 0,
  status: 'healthy',
  yearlyLimit: limit * 12,
  yearlySpent: spent,
  yearlyRemaining: limit * 12 - spent,
});

// Invented figures.
const report: BudgetReport = {
  month: '2026-09',
  daysInMonth: 30,
  daysElapsed: 30,
  daysRemaining: 0,
  totalLimit: 300,
  totalSpent: 150,
  totalRemaining: 150,
  percentUsed: 50,
  dailyBurn: 5,
  projectedSpend: 150,
  status: 'healthy',
  lines: [line('Groceries', 100, 200), line('Transport', 50, 100)],
};

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Budget page', () => {
  it('Given the report is still loading When the page renders Then the header and month switcher are usable at once', () => {
    render(BudgetPage, {
      data: { engine: never(), month: '2026-09', report: never(), transactions: never(), categories: never() } as never,
    });

    expect(screen.getByRole('heading', { level: 1, name: 'Budget' })).toBeTruthy();
    expect(screen.getByText('Sep 2026')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Previous month' }).getAttribute('title')).toBe('Previous month');
    expect(screen.getByRole('button', { name: 'Next month' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Move money' }).hasAttribute('disabled')).toBe(true);
    expect(screen.getByText(/Loading the budget…/)).toBeTruthy();
    expect(screen.getByText('Loading categories…')).toBeTruthy();
  });

  it('Given the report arrives before the transactions When it settles Then the overview and categories show', async () => {
    render(BudgetPage, {
      data: {
        engine: never(),
        month: '2026-09',
        report: settled(report),
        transactions: never(),
        categories: never(),
      } as never,
    });
    await flush();

    expect(screen.getByText('Am I within budget?')).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Edit limit for Groceries' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Move money' }).hasAttribute('disabled')).toBe(false);
    expect(screen.getByText('Loading categories…')).toBeTruthy();
  });
});
