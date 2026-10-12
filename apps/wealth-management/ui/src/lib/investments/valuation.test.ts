import { describe, expect, it } from 'vitest';
import type { Holding } from '$lib/api/types';
import { sectionTotal, toVND } from './valuation';

const holding = (value: number | null, currency: Holding['currency']): Holding => ({
  platform: 'P',
  symbol: 'S',
  quantity: null,
  price: null,
  currency,
  value,
  valueSource: 'sheet',
});

describe('valuation', () => {
  it('converts with known rates only', () => {
    expect(toVND(2, 'USD', { USD: 25_000 })).toBe(50_000);
    expect(toVND(2, 'USDT', { USD: 25_000 })).toBeNull();
    expect(toVND(7, 'VND', {})).toBe(7);
  });

  it('totals what it can and counts the rest as unvalued', () => {
    const total = sectionTotal([holding(1_000, 'VND'), holding(2, 'USD'), holding(3, 'USDT'), holding(null, 'VND')], {
      USD: 25_000,
    });
    expect(total).toEqual({ vnd: 51_000, unvalued: 2, byCurrency: { VND: 1_000, USD: 2, USDT: 3 } });
  });
});
