import type { FundPerformance, FundReturn } from '$lib/api/types';
import { describe, expect, it } from 'vitest';
import { depositRateLabel, formatPoints, fundHeadline, VS_DEPOSIT } from './fundPerformance';

const total: FundReturn = {
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
};

const performance = (overrides: Partial<FundPerformance> = {}): FundPerformance => ({
  holdings: [],
  funds: [],
  totals: { holdings: 1, costKnown: 1, costUnknown: 0, value: 110_000, valueCostKnown: 110_000, return: total },
  deposit: { bank: 'Bank A', tenor: '', rate: 5, basis: 'best published savings rate today' },
  generatedAt: '2026-01-02T00:00:00Z',
  ...overrides,
});

describe('fundHeadline', () => {
  it('compares with the deposit when the cost and a rate are known', () => {
    expect(fundHeadline(performance())).toEqual({ kind: 'compare', total, rate: 5, vs: 'ahead' });
  });

  it('says the comparison is missing without a deposit rate', () => {
    const headline = fundHeadline(
      performance({
        deposit: null,
        depositIssue: 'bank rates unavailable',
        totals: { ...performance().totals, return: { ...total, depositValue: null } },
      }),
    );
    expect(headline).toEqual({
      kind: 'no_deposit',
      total: { ...total, depositValue: null },
      issue: 'bank rates unavailable',
    });
  });

  it('has no answer without a known cost or without holdings', () => {
    expect(fundHeadline(performance({ totals: { ...performance().totals, return: null } })).kind).toBe('no_cost');
    expect(fundHeadline(performance({ totals: { ...performance().totals, holdings: 0, return: null } })).kind).toBe(
      'empty',
    );
  });
});

describe('labels', () => {
  it('names the rate used and signs the points', () => {
    expect(depositRateLabel(performance())).toBe('Bank A · 5.00% a year');
    expect(depositRateLabel(performance({ deposit: null }))).toBeNull();
    expect(formatPoints(1.234)).toBe('+1.23 points a year');
    expect(formatPoints(-0.5)).toBe('-0.50 points a year');
    expect(formatPoints(null)).toBe('-');
  });

  it('gives every comparison a status tone and label', () => {
    expect(VS_DEPOSIT.ahead.tone).toBe('good');
    expect(VS_DEPOSIT.behind.tone).toBe('warning');
    expect(VS_DEPOSIT.even.label).toBe('About even');
  });
});
