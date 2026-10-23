import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { CashFlow, YearlyCashFlow } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';

// Sections that load on their own (recommendations, chart insight) stay pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
const goto = vi.hoisted(() => vi.fn());
vi.mock('$app/navigation', () => ({ goto, invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$app/state', () => ({ page: { url: new URL('http://127.0.0.1:5173/ui/cash-flow?year=2025') } }));

import CashFlowPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented figures.
const flow = (month: string, income: number, expense: number): CashFlow => ({
  month,
  income,
  expense,
  cashFlow: income - expense,
  savingsRate: income ? ((income - expense) / income) * 100 : 0,
});
const months = Array.from({ length: 12 }, (_, i) =>
  flow(`2025-${String(i + 1).padStart(2, '0')}`, 31_400_000, 27_300_000),
);
const none = { category: '', values: [], total: 0 };
const report: YearlyCashFlow = {
  year: 2025,
  years: [2026, 2025, 2024],
  monthsElapsed: 12,
  months,
  total: flow('2025', 376_800_000, 327_600_000),
  monthlyAverage: flow('2025', 31_400_000, 27_300_000),
  previousYear: flow('2024', 0, 0),
  incomeChangePercent: null,
  expenseChangePercent: null,
  income: [],
  expense: [],
  verdict: { status: 'healthy', savingsRate: 13, deficitMonths: 0 },
  projection: null,
  spikes: [],
  savingsTarget: { watch: 5, healthy: 10, target: 20 },
  expenseSplit: {
    fixed: none,
    essential: none,
    flexible: none,
    untagged: none,
    needsShareOfIncome: null,
    needsShareComfortable: 50,
    wantsShareOfIncome: null,
    savingsShareOfIncome: null,
    savingsShareTarget: 30,
    untaggedCategories: [],
    room: [],
    roomPerMonth: 0,
    roomPerYear: 0,
  },
};

describe('Cash flow page', () => {
  afterEach(() => mask.hide());

  it('Given the report is still loading When the page renders Then the header and year switcher are usable at once', async () => {
    render(CashFlowPage, { data: { engine: never(), year: 2025, report: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Cash Flow' })).toBeTruthy();
    expect(screen.getByText('Income and expense by month, 2025')).toBeTruthy();
    expect(screen.getByRole<HTMLSelectElement>('combobox', { name: 'Year' }).value).toBe('2025');
    expect(screen.getByRole('button', { name: 'Previous year' }).getAttribute('title')).toBe('Previous year');
    expect(screen.getAllByText('Loading the cash flow…').length).toBeGreaterThan(0);
    expect(screen.getByText('Am I okay? 2025')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Previous year' }));
    expect(String(goto.mock.calls.at(-1)?.[0])).toContain('year=2024');
  });

  it('Given the report fails When it settles Then the error and Retry show in its place under the header', async () => {
    render(CashFlowPage, {
      data: { engine: never(), year: 2025, report: defer(Promise.reject(new Error('ledger tab missing'))) } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('ledger tab missing');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('heading', { level: 1, name: 'Cash Flow' })).toBeTruthy();
  });

  it('Given the report has loaded When amounts are masked Then the totals stay hidden until toggled', async () => {
    render(CashFlowPage, { data: { engine: never(), year: 2025, report: settled(report) } as never });
    await flush();

    expect(screen.getByText('Am I okay? 2025')).toBeTruthy();
    expect(document.body.textContent).not.toMatch(/376\.800\.000|31\.400\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toMatch(/376\.800\.000/);
  });
});
