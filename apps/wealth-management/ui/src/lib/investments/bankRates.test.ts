import { describe, expect, it } from 'vitest';
import { bestBankRate } from './bankRates';

describe('bestBankRate', () => {
  it('picks the highest published rate and ignores zero rates', () => {
    const rates = [
      { bank: 'Bank A', tenor: '6 months', rate: 4.5 },
      { bank: 'Bank B', tenor: '12 months', rate: 5.9 },
      { bank: 'Bank C', tenor: '', rate: 0 },
    ];
    expect(bestBankRate(rates)).toEqual({ bank: 'Bank B', tenor: '12 months', rate: 5.9 });
  });

  it('is null when no bank published a rate', () => {
    expect(bestBankRate([])).toBeNull();
    expect(bestBankRate([{ bank: 'Bank A', tenor: '', rate: 0 }])).toBeNull();
  });
});
