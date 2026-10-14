import { describe, expect, it } from 'vitest';
import type { ExchangeRate } from './api/types';
import { rateDetail } from './rates';

const rate = (patch: Partial<ExchangeRate>): ExchangeRate => ({
  from: 'USD',
  to: 'VND',
  rate: 26300,
  provider: 'fmarket',
  ...patch,
});

describe('rateDetail', () => {
  it('says what kind of rate a provider quoted', () => {
    expect(rateDetail(rate({ basis: 'bank_sell' }))).toBe('bank sell');
    expect(rateDetail(rate({ provider: 'yahoo', basis: 'market' }))).toBe('market');
    expect(rateDetail(rate({ provider: 'currencyapi', basis: 'reference' }))).toBe('daily reference');
    expect(rateDetail(rate({ provider: 'coingecko', basis: 'crypto_market' }))).toBe('crypto market');
  });

  it("adds the provider's date", () => {
    expect(rateDetail(rate({ basis: 'bank_sell', timestamp: '2026-10-14T00:00:00Z' }))).toBe('bank sell · 14/10/2026');
    expect(rateDetail(rate({ timestamp: '2026-10-14' }))).toBe('14/10/2026');
  });

  it('is undefined when neither is known', () => {
    expect(rateDetail(rate({}))).toBeUndefined();
  });
});
