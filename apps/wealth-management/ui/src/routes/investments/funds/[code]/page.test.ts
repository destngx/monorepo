import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { FundDetail } from '$lib/investments/fmarket';

const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/paths', () => ({ base: '/ui' }));

import FundPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// An invented fund.
const fund: FundDetail = {
  id: 7,
  code: 'AAA',
  name: 'AAA Growth Fund',
  issuer: 'Example Capital',
  issuerLogo: null,
  assetType: 'Stock',
  assetCode: 'BOND',
  nav: 12_345.67,
  return12m: 12,
  annualized36m: 8,
  updatedAt: '2026-10-15',
  description: 'Invests in listed shares.',
  riskLevel: 'High',
  inceptionDate: '2020-01-01',
  managementFee: 1.5,
  performanceFee: null,
  buyMinValue: 10_000,
  sellMinUnits: 1,
  holdingMinUnits: 1,
  orderCutoff: '14:40',
  nextTradingDate: '2026-10-20',
  website: null,
  holdingsAsOf: null,
  changes: { navTo12Months: 12 },
  fees: [],
  topHoldings: [],
  allocation: [],
  industries: [],
};

describe('Fund detail page', () => {
  it('Given the detail is still loading When the page renders Then the code and the back link show at once', () => {
    render(FundPage, { data: { code: 'AAA', detail: never(), nav: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'AAA' })).toBeTruthy();
    expect(screen.getByRole('link', { name: /All funds/ }).getAttribute('href')).toBe('/ui/investments/funds');
    expect(screen.getByText('Loading NAV per unit…')).toBeTruthy();
  });

  it('Given the detail arrives but the NAV history fails When it settles Then the fund shows and only the chart has the error and Retry', async () => {
    render(FundPage, {
      data: {
        code: 'AAA',
        detail: settled({ fund }),
        nav: defer(Promise.reject(new Error('NAV history timed out'))),
      } as never,
    });
    await flush();

    expect(screen.getByText('AAA Growth Fund')).toBeTruthy();
    expect(screen.getByText('Overview')).toBeTruthy();
    expect(screen.getByRole('link', { name: /All funds/ }).getAttribute('href')).toBe(
      '/ui/investments/funds?type=BOND',
    );
    expect(screen.getByText('Could not load the NAV history')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(navigation.invalidateAll).toHaveBeenCalled();
  });

  it('Given an unknown code When the detail fails Then the section explains it instead of the whole page failing', async () => {
    render(FundPage, {
      data: {
        code: 'ZZZ',
        detail: defer(Promise.reject(new Error('Fund ZZZ was not found on fmarket.'))),
        nav: defer(Promise.reject(new Error('Fund ZZZ was not found on fmarket.'))),
      } as never,
    });
    await flush();

    expect(screen.getByRole('heading', { level: 1, name: 'ZZZ' })).toBeTruthy();
    expect(screen.getByText('Could not load fund ZZZ')).toBeTruthy();
    expect(screen.getAllByRole('alert')).toHaveLength(1);
  });
});
