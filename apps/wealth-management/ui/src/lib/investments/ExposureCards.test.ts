import type { CryptoExposure, Exposure, HedgeTrend, Hedges } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import CryptoCard from './CryptoCard.svelte';
import ExposureCard from './ExposureCard.svelte';
import HedgesCard from './HedgesCard.svelte';

// Invented figures only.
const crypto = (overrides: Partial<CryptoExposure> = {}): CryptoExposure => ({
  value: 100_000_000,
  holdings: 100_000_000,
  accounts: 40_000_000,
  share: 50,
  shareOfNetWorth: 66.67,
  coins: [
    {
      symbol: 'BTC',
      quantity: 1,
      price: { price: 2_660, currency: 'USD', provider: 'coingecko' },
      value: 70_000_000,
      unvalued: 0,
      share: 70,
      move: { basis: '24h', price: 2_660, previous: 1_900, changePct: 40, asOf: '', since: '' },
      change: 20_000_000,
    },
    { symbol: 'ETH', quantity: 2, value: 30_000_000, unvalued: 0, share: 30, change: null, issue: 'no bars' },
  ],
  change: 20_000_000,
  scenarios: [
    { fallPct: 30, loss: 30_000_000, netWorthAfter: 120_000_000, netWorthChangePct: -20 },
    { fallPct: 50, loss: 50_000_000, netWorthAfter: 100_000_000, netWorthChangePct: -33.33 },
  ],
  ...overrides,
});

const exposure = (overrides: Partial<Exposure> = {}): Exposure => ({
  total: 200_000_000,
  netWorth: 150_000_000,
  volatileValue: 130_000_000,
  volatileShare: 65,
  change: 30_000_000,
  changePct: 42.86,
  covered: 100_000_000,
  classes: [
    {
      class: 'cash',
      label: 'Cash and deposits',
      value: 60_000_000,
      share: 30,
      volatile: false,
      change: null,
      changePct: null,
      covered: 0,
    },
    {
      class: 'funds',
      label: 'Fund certificates',
      value: 30_000_000,
      share: 15,
      volatile: true,
      change: 10_000_000,
      changePct: 50,
      covered: 30_000_000,
    },
    {
      class: 'crypto',
      label: 'Crypto',
      value: 100_000_000,
      share: 50,
      volatile: true,
      change: 20_000_000,
      changePct: 40,
      covered: 70_000_000,
    },
    {
      class: 'moneyLent',
      label: 'Money lent',
      value: 10_000_000,
      share: 5,
      volatile: false,
      change: null,
      changePct: null,
      covered: 0,
    },
  ],
  crypto: crypto(),
  funds: [],
  issues: [],
  generatedAt: '2026-10-17T00:00:00Z',
  ...overrides,
});

const trend = (overrides: Partial<HedgeTrend> = {}): HedgeTrend => ({
  asset: 'gold',
  label: 'SJC gold',
  symbol: 'SJC',
  unit: 'VND per tael (SJC sell price)',
  latest: 120,
  latestDate: '2026-10-16',
  changes: [
    { period: '3M', fromDate: '2026-07-16', from: 110, changePct: 9.09, partial: false, onCash: 9_090_000 },
    { period: '1Y', fromDate: '2025-10-16', from: 100, changePct: 20, partial: false, onCash: 20_000_000 },
  ],
  points: [
    { date: '2025-10-16', close: 100 },
    { date: '2026-07-16', close: 110 },
    { date: '2026-10-16', close: 120 },
  ],
  ...overrides,
});

const text = (id: string) => screen.getByTestId(id).textContent?.replace(/\s+/g, ' ') ?? '';

afterEach(() => mask.hide());

describe('ExposureCard', () => {
  it('leads with the volatile share and today’s move, then each class', () => {
    mask.toggle();
    render(ExposureCard, { exposure: exposure() });

    expect(text('exposure-headline')).toMatch(
      /65% of your assets move with markets; today they moved \+30[.,]000[.,]000/,
    );
    expect(text('exposure-headline')).toContain('(+42.86%)');
    expect(screen.getAllByText('Moves with markets').length).toBe(2);
    expect(screen.getAllByText('Fixed VND').length).toBe(2);
    expect(screen.getByText(/Today's change covers/)).toBeTruthy();
  });

  it('says so when no price moved today, and keeps amounts masked', () => {
    render(ExposureCard, { exposure: exposure({ change: null, changePct: null, covered: 0 }) });

    expect(text('exposure-headline')).toContain("today's change is not available");
    expect(screen.queryByText(/30[.,]000[.,]000/)).toBeNull();
  });
});

describe('CryptoCard', () => {
  it('answers with the share of net worth and the 30% scenario, graded with an icon and label', () => {
    mask.toggle();
    const { container } = render(CryptoCard, { crypto: crypto(), netWorth: 150_000_000 });

    const headline = text('crypto-headline');
    expect(headline).toContain('66.7% of your net worth');
    expect(headline).toMatch(/If crypto falls 30%, net worth falls by 30[.,]000[.,]000/);
    expect(headline).toContain('(-20.0%)');
    expect(container.querySelectorAll('[data-status="warning"]').length).toBe(2);
    expect(screen.getByText('Large hit')).toBeTruthy();
    expect(screen.getByText('+40.00%')).toBeTruthy();
    expect(screen.getByText('no change data')).toBeTruthy();
  });

  it('shows an empty state without crypto', () => {
    render(CryptoCard, { crypto: crypto({ value: 0, coins: [], scenarios: [] }), netWorth: 1 });
    expect(screen.getByText('No crypto holdings or exchange balances.')).toBeTruthy();
  });
});

describe('HedgesCard', () => {
  const hedges = (overrides: Partial<Hedges> = {}): Hedges => ({
    cash: 100_000_000,
    cashShare: 50,
    vndShare: 75,
    gold: trend(),
    usd: trend({ asset: 'usd', label: 'US dollar', symbol: 'USDVND', unit: 'VND per USD' }),
    issues: [],
    generatedAt: '2026-10-17T00:00:00Z',
    ...overrides,
  });

  it('frames gold and USD against the cash share, with a legend for both series', () => {
    render(HedgesCard, { hedges: hedges() });

    expect(text('hedges-headline')).toBe(
      'Your cash is 50% of your assets, all in VND; gold rose 20.0% and the US dollar rose 20.0% over 12 months.',
    );
    expect(screen.getByLabelText('Legend').textContent).toContain('SJC gold');
    expect(screen.getByRole('img', { name: /percent change over 1Y/ })).toBeTruthy();
  });

  it('switches the chart range and lists a missing trend as an issue', async () => {
    render(HedgesCard, { hedges: hedges({ usd: null, issues: ['US dollar trend unavailable: upstream down'] }) });

    await fireEvent.click(screen.getByRole('button', { name: '3M' }));
    expect(screen.getByRole('img', { name: /percent change over 3M/ })).toBeTruthy();
    expect(screen.queryByLabelText('Legend')).toBeNull();
    expect(screen.getByText('US dollar trend unavailable: upstream down')).toBeTruthy();
  });
});
