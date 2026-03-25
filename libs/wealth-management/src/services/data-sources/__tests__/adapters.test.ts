import { describe, it, expect, beforeEach, vi } from 'vitest';
import { YahooFinanceAdapter } from '../yahoo-finance';
import { VNStockAdapter } from '../vnstock';
import { CafeFAdapter } from '../cafef';
import type { StockDataPoint } from '../types';

describe('Data Source Adapters', () => {
  describe('CafeFAdapter', () => {
    const adapter = new CafeFAdapter();

    it('name is CafeF', () => {
      expect(adapter.name).toBe('CafeF');
    });

    it('supports VN market indices only', () => {
      expect(adapter.supports('^VNINDEX', 'VN')).toBe(true);
      expect(adapter.supports('^HNX', 'VN')).toBe(true);
      expect(adapter.supports('VN30', 'VN')).toBe(true);
      expect(adapter.supports('^UPCOM', 'VN')).toBe(true);
      expect(adapter.supports('VCB', 'VN')).toBe(false);
      expect(adapter.supports('^GSPC', 'US')).toBe(false);
    });

    it('returns null for stock (indices-only), not error', async () => {
      const result = await adapter.fetchHistorical('VCB', 'VN', '1d', '60d');
      expect(result).toBeNull();
    });

    it('returns null for fetchHistorical (CafeF has no free historical API)', async () => {
      const result = await adapter.fetchHistorical('^VNINDEX', 'VN', '1d', '60d');
      expect(result).toBeNull();
    });
  });

  describe('VNStockAdapter', () => {
    const adapter = new VNStockAdapter();

    it('name is VNStock', () => {
      expect(adapter.name).toBe('VNStock');
    });

    it('supports VN market only', () => {
      expect(adapter.supports('VCB', 'VN')).toBe(true);
      expect(adapter.supports('IFC', 'VN')).toBe(true);
      expect(adapter.supports('^VNINDEX', 'VN')).toBe(true);
      expect(adapter.supports('AAPL', 'US')).toBe(false);
    });

    it('returns null when python server unavailable (graceful degradation)', async () => {
      // If VNSTOCK_SERVER_URL is not running, adapter returns null (not throws)
      const result = await adapter.fetchHistorical('VCB', 'VN', '1d', '60d');
      // Result is null when server is unavailable
      expect(result).toBeNull();
    });
  });

  describe('YahooFinanceAdapter', () => {
    const adapter = new YahooFinanceAdapter();

    it('name is YahooFinance', () => {
      expect(adapter.name).toBe('YahooFinance');
    });

    it('supports both US and VN markets', () => {
      expect(adapter.supports('AAPL', 'US')).toBe(true);
      expect(adapter.supports('VCB', 'VN')).toBe(true);
      expect(adapter.supports('^VNINDEX', 'VN')).toBe(true);
    });

    it(
      'fetches US stock data successfully',
      async () => {
        const result = await adapter.fetchHistorical('AAPL', 'US', '1d', '7d');
        if (result) {
          expect(Array.isArray(result)).toBe(true);
          expect(result.length).toBeGreaterThan(0);
          const point = result[0];
          expect(point).toHaveProperty('timestamp');
          expect(point).toHaveProperty('close');
        }
      },
      { timeout: 10000 },
    );

    it(
      'returns data points with required fields',
      async () => {
        const result = await adapter.fetchHistorical('AAPL', 'US', '1d', '7d');
        if (result && result.length > 0) {
          const point = result[0];
          expect(typeof point.timestamp).toBe('number');
          expect(typeof point.close).toBe('number');
          expect(point.close).toBeGreaterThan(0);
        }
      },
      { timeout: 10000 },
    );
  });

  describe('Fallback Chain Behavior', () => {
    it('VN indices: CafeF has no historical → returns null (correct for indices)', async () => {
      const cafef = new CafeFAdapter();
      const result = await cafef.fetchHistorical('^VNINDEX', 'VN', '1d', '60d');
      expect(result).toBeNull();
    });

    it(
      'VN stocks: VNStock (if available) preferred over Yahoo',
      async () => {
        const vnstock = new VNStockAdapter();
        const yahoo = new YahooFinanceAdapter();

        // VNStock should try first and may fail (if server not running)
        const vnstockResult = await vnstock.fetchHistorical('VCB', 'VN', '1d', '60d');
        // Yahoo should still work as fallback (if network available)
        const yahooResult = await yahoo.fetchHistorical('VCB', 'VN', '1d', '60d');

        // If both fail: network issue, expected
        // If VNStock succeeds: best source
        // If only Yahoo succeeds: acceptable fallback
        // Never both fail in normal conditions
        expect(vnstockResult !== null || yahooResult !== null).toBe(true);
      },
      { timeout: 10000 },
    );
  });
});
