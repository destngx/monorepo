import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { FundSummary, FundType, Sourced } from '$lib/investments/fmarket';

// The gold tab loads on its own; its calls stay pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
const navigation = vi.hoisted(() => ({ goto: vi.fn(), invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/paths', () => ({ base: '/ui' }));
const appState = vi.hoisted(() => ({ page: { url: new URL('http://127.0.0.1:5173/ui/investments/funds') } }));
vi.mock('$app/state', () => appState);

import FundsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented funds.
const fund = (id: number, code: string, return12m: number): FundSummary => ({
  id,
  code,
  name: `${code} Growth Fund`,
  issuer: 'Example Capital',
  issuerLogo: null,
  assetType: 'Stock',
  assetCode: 'STOCK',
  nav: 12_345.67,
  return12m,
  annualized36m: 8,
  updatedAt: '2026-10-15',
});

type Lists = Record<FundType, Streamed<Sourced<FundSummary[]>>>;
const lists = (overrides: Partial<Lists> = {}): Lists => ({
  STOCK: never(),
  BOND: never(),
  BALANCED: never(),
  MMF: never(),
  ...overrides,
});

describe('Fmarket funds page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    appState.page.url = new URL('http://127.0.0.1:5173/ui/investments/funds');
  });

  it('Given every list is still loading When the page renders Then the tabs and fund filters are usable at once', async () => {
    render(FundsPage, { data: { funds: lists(), bankRates: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Fmarket' })).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Bank rates' })).toBeTruthy();
    const filters = screen.getByRole('group', { name: 'Fund type' });
    expect(within(filters).getByRole('button', { name: 'Bond' }).getAttribute('aria-busy')).toBe('true');
    expect(screen.getByText('Loading stock funds…')).toBeTruthy();

    await fireEvent.click(within(filters).getByRole('button', { name: 'Bond' }));
    expect(navigation.goto).toHaveBeenCalled();
  });

  it('Given the stock list arrives When it settles Then its rows open the fund and the filter shows the count', async () => {
    render(FundsPage, {
      data: {
        funds: lists({ STOCK: settled({ data: [fund(1, 'AAA', 12), fund(2, 'BBB', 4)] }) }),
        bankRates: never(),
      } as never,
    });
    await flush();

    const link = screen.getByRole('link', { name: /AAA/ });
    expect(link.getAttribute('href')).toBe('/ui/investments/funds/AAA');
    expect(link.closest('tr')?.classList.contains('row-link')).toBe(true);
    expect(
      within(screen.getByRole('group', { name: 'Fund type' })).getByRole('button', { name: /^Stock\s*2$/ }),
    ).toBeTruthy();
  });

  it('Given the stock list fails When the others load Then only its table shows the error with a Retry', async () => {
    render(FundsPage, {
      data: {
        funds: lists({
          STOCK: defer(Promise.reject(new Error('fmarket timed out'))),
          BOND: settled({ data: [fund(3, 'CCC', 2)] }),
        }),
        bankRates: never(),
      } as never,
    });
    await flush();

    expect(screen.getByText('Could not load stock funds')).toBeTruthy();
    expect(screen.getByText('fmarket timed out')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(navigation.invalidateAll).toHaveBeenCalled();
    expect(
      within(screen.getByRole('group', { name: 'Fund type' })).getByRole('button', { name: /^Bond\s*1$/ }),
    ).toBeTruthy();
  });

  it('Given the bank tab When the rates are still loading Then the card title shows with a skeleton', () => {
    appState.page.url = new URL('http://127.0.0.1:5173/ui/investments/funds?tab=bank');
    render(FundsPage, { data: { funds: lists(), bankRates: never() } as never });

    expect(screen.getByText('Bank savings rates')).toBeTruthy();
    expect(screen.getByText('Loading bank rates…')).toBeTruthy();
  });
});
