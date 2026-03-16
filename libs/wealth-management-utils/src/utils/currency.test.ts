import { describe, it, expect } from 'vitest';
import { formatVND, formatUSD, convertUSDTtoVND } from './currency';

describe('Currency Utils', () => {
  describe('formatVND', () => {
    it('properly formats VND currency', () => {
      // Depending on node environment, it may format as "100.000\xa0₫"
      const result = formatVND(100000);
      expect(result).toMatch(/100\.000/);
      expect(result).toMatch(/₫/);
    });
  });

  describe('formatUSD', () => {
    it('properly formats USD currency', () => {
      const result = formatUSD(100.50);
      expect(result).toBe('$100.50');
    });

    it('formats USD with integers', () => {
      const result = formatUSD(500);
      expect(result).toBe('$500.00');
    });
  });

  describe('convertUSDTtoVND', () => {
    it('converts correctly', () => {
      const result = convertUSDTtoVND(10, 25000);
      expect(result).toBe(250000);
    });
  });
});
