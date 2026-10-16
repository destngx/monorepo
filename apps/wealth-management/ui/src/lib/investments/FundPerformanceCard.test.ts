import type { FundHoldingPerformance, FundPerformance, FundReturn } from '$lib/api/types';
import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import FundPerformanceCard from './FundPerformanceCard.svelte';

const fundReturn = (overrides: Partial<FundReturn> = {}): FundReturn => ({
  totalBought: 100_000,
  totalSold: 0,
  netInvested: 100_000,
  value: 110_000,
  gain: 10_000,
  simpleReturn: 10,
  annualisedReturn: 10,
  firstDate: '2025-01-01',
  asOf: '2026-01-01',
  depositValue: 105_000,
  depositGain: 5_000,
  differenceVND: 5_000,
  differencePoints: 5,
  vsDeposit: 'ahead',
  ...overrides,
});

const holding = (overrides: Partial<FundHoldingPerformance> = {}): FundHoldingPerformance => ({
  platform: 'Platform A',
  symbol: 'FUND-A',
  units: 10,
  historyUnits: 10,
  nav: 11_000,
  navDate: '2026-01-01',
  value: 110_000,
  costStatus: 'known',
  costBasis: 100_000,
  unrealisedGain: 10_000,
  cashFlows: [],
  return: fundReturn(),
  issues: [],
  ...overrides,
});

const performance = (overrides: Partial<FundPerformance> = {}): FundPerformance => ({
  holdings: [
    holding(),
    holding({
      symbol: 'FUND-B',
      units: 5,
      historyUnits: 2,
      value: 50_000,
      costStatus: 'units_mismatch',
      costBasis: null,
      unrealisedGain: null,
      return: null,
      issues: ['History adds up to 2 units but Total Unit is 5'],
    }),
  ],
  funds: [],
  totals: { holdings: 2, costKnown: 1, costUnknown: 1, value: 160_000, valueCostKnown: 110_000, return: fundReturn() },
  deposit: { bank: 'Bank A', tenor: '', rate: 5, basis: 'best published savings rate today' },
  generatedAt: '2026-01-02T00:00:00Z',
  ...overrides,
});

describe('FundPerformanceCard', () => {
  it('leads with the answer: the annual return, the deposit rate, what it would have earned and the difference', () => {
    render(FundPerformanceCard, { performance: performance() });

    const headline = screen.getByTestId('fund-headline').textContent?.replace(/\s+/g, ' ') ?? '';
    expect(headline).toContain('Your funds earned 10.00% a year');
    expect(headline).toContain('a deposit at 5.00% would have earned');
    expect(headline).toContain('+5.00 points a year');
    expect(screen.getByText(/Deposit: Bank A · 5.00% a year/)).toBeTruthy();
  });

  it('shows ahead or behind with a status icon and label, and why a cost is unknown', () => {
    const { container } = render(FundPerformanceCard, { performance: performance() });

    expect(container.querySelectorAll('[data-status="good"]').length).toBe(2);
    expect(screen.getAllByText('Ahead of deposit').length).toBe(2);
    expect(screen.getByText('Cost unknown: History ≠ Total Unit')).toBeTruthy();
    expect(screen.getByText(/History adds up to 2 units/)).toBeTruthy();
    expect(screen.getByText(/Covers 1 of 2 holdings/)).toBeTruthy();
  });

  it('says so when no holding has a known cost', () => {
    render(FundPerformanceCard, {
      performance: performance({ totals: { ...performance().totals, costKnown: 0, return: null } }),
    });
    expect(screen.getByTestId('fund-headline').textContent).toContain('Cost is unknown for every fund holding');
  });

  it('marks a holding behind the deposit as a warning', () => {
    const behind = fundReturn({ differenceVND: -3_000, differencePoints: -2, vsDeposit: 'behind' });
    const { container } = render(FundPerformanceCard, {
      performance: performance({
        holdings: [holding({ return: behind })],
        totals: { ...performance().totals, return: behind },
      }),
    });
    expect(container.querySelectorAll('[data-status="warning"]').length).toBe(2);
    expect(screen.getAllByText('Behind deposit').length).toBe(2);
  });
});
