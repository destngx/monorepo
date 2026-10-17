import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { PortfolioValuation } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          return new Promise(() => undefined);
        },
    },
  ),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import InvestmentsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const pendingData = () => ({
  engine: never(),
  valuation: never(),
  allocation: never(),
  bankRates: never(),
  fundPerformance: never(),
  exposure: never(),
  hedges: never(),
});

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented figures.
const valuation: PortfolioValuation = {
  holdings: [],
  rates: [],
  totals: {
    vndAccounts: 4_100_000,
    funds: 1_000_000,
    vnd: 5_100_000,
    cryptoAccounts: 2_300_000,
    cryptoHoldings: 2_700_000,
    crypto: 2_700_000,
    total: 7_800_000,
    unvalued: 0,
  },
  issues: [],
  generatedAt: '2026-10-15T08:00:00Z',
};

describe('Investments page', () => {
  afterEach(() => mask.hide());

  it('Given none of the six calls has answered When the page renders Then the shell and a skeleton per section show', () => {
    render(InvestmentsPage, { data: pendingData() as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Investments' })).toBeTruthy();
    expect(screen.getByRole('button', { name: /Refresh prices/ })).toBeTruthy();
    expect(screen.getByRole('link', { name: 'All bank rates' })).toBeTruthy();
    const loading = screen.getAllByRole('status').map((status) => status.textContent?.trim());
    expect(loading).toEqual(
      expect.arrayContaining([
        'Loading my exposure…',
        'Loading gold and USD as protection…',
        'Loading portfolio total…',
        'Loading allocation…',
        'Loading funds vs a bank deposit…',
        'Loading bank rates…',
      ]),
    );
  });

  it('Given the valuation fails When it settles Then its sections show the error with Retry and the rest keep loading', async () => {
    render(InvestmentsPage, {
      data: { ...pendingData(), valuation: defer(Promise.reject(new Error('prices sheet missing'))) } as never,
    });
    await flush();

    const alerts = screen.getAllByRole('alert').map((alert) => alert.textContent ?? '');
    expect(alerts.some((text) => text.includes('Could not load portfolio totals'))).toBe(true);
    expect(alerts.some((text) => text.includes('Could not load holdings'))).toBe(true);
    expect(alerts.every((text) => text.includes('prices sheet missing'))).toBe(true);
    expect(screen.getAllByRole('button', { name: /Retry/ }).length).toBe(2);
    expect(screen.getByText('Loading my exposure…')).toBeTruthy();
  });

  it('Given the page When Refresh prices is clicked Then it forces fresh figures and stays busy until they answer', async () => {
    render(InvestmentsPage, { data: pendingData() as never });
    calls.length = 0;

    const button = screen.getByRole('button', { name: /Refresh prices/ });
    await fireEvent.click(button);
    await flush();

    expect(calls.map((call) => call.name).sort()).toEqual([
      'allocation',
      'exposure',
      'fundPerformance',
      'hedges',
      'valuation',
    ]);
    expect(calls.every((call) => (call.args[0] as { force?: boolean }).force)).toBe(true);
    expect(button.getAttribute('aria-busy')).toBe('true');
  });

  it('Given the totals have loaded When amounts are masked Then the stat hints hide their amounts too', async () => {
    render(InvestmentsPage, { data: { ...pendingData(), valuation: settled(valuation) } as never });
    await flush();

    const vndHint = screen.getByTestId('vnd-hint').textContent ?? '';
    const cryptoHint = screen.getByTestId('crypto-hint').textContent ?? '';
    expect(vndHint).toBe(`Accounts ${MASK} + funds`);
    expect(cryptoHint).toBe(`Max of accounts ${MASK} and holdings ${MASK}`);
    expect(document.body.textContent).not.toMatch(/4\.100\.000|2\.300\.000|2\.700\.000/);

    mask.toggle();
    await flush();
    expect(screen.getByTestId('vnd-hint').textContent).toContain('4.100.000');
    expect(screen.getByTestId('crypto-hint').textContent).toContain('2.300.000');
  });
});
