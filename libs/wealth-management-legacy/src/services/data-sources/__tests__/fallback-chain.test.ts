import { describe, it, expect, vi } from 'vitest';
import type { DataSourceAdapter, StockDataPoint } from '../../data-sources';

describe('Multi-Source Fallback Chain (VNStock → Yahoo → CafeF)', () => {
  // Mock adapters for testing fallback logic
  const createMockAdapter = (
    name: string,
    shouldFail: boolean,
    data: StockDataPoint[] | null = null,
  ): DataSourceAdapter => ({
    name,
    supports: () => true,
    fetchStock: async () => null,
    fetchHistorical: async () => {
      if (shouldFail) return null;
      return (
        data || [
          { timestamp: Math.floor(Date.now() / 1000) - 86400, close: 100, high: 105, low: 95 },
          { timestamp: Math.floor(Date.now() / 1000), close: 102, high: 106, low: 96 },
        ]
      );
    },
  });

  it('tries adapters in sequence: VNStock → Yahoo → CafeF', async () => {
    const adapters = [
      createMockAdapter('VNStock', true), // VNStock fails
      createMockAdapter('Yahoo', true), // Yahoo fails
      createMockAdapter('CafeF', false), // CafeF succeeds (last resort)
    ];

    let successSource = '';
    for (const adapter of adapters) {
      const result = await adapter.fetchHistorical('TEST', 'VN', '1d', '60d');
      if (result && result.length > 0) {
        successSource = adapter.name;
        break;
      }
    }

    expect(successSource).toBe('CafeF');
  });

  it('stops at VNStock when it succeeds (no unnecessary requests)', async () => {
    const calls = { vnstock: 0, yahoo: 0, cafef: 0 };

    const adapters = [
      {
        name: 'VNStock',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.vnstock++;
          return [{ timestamp: Math.floor(Date.now() / 1000), close: 100 }];
        },
      } as DataSourceAdapter,
      {
        name: 'Yahoo',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.yahoo++;
          return [{ timestamp: Math.floor(Date.now() / 1000), close: 100 }];
        },
      } as DataSourceAdapter,
      {
        name: 'CafeF',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.cafef++;
          return [{ timestamp: Math.floor(Date.now() / 1000), close: 100 }];
        },
      } as DataSourceAdapter,
    ];

    for (const adapter of adapters) {
      if (!adapter.supports('TEST', 'VN')) continue;
      const result = await adapter.fetchHistorical('TEST', 'VN', '1d', '60d');
      if (result && result.length > 0) {
        break;
      }
    }

    expect(calls.vnstock).toBe(1); // Called once, succeeded
    expect(calls.yahoo).toBe(0); // Not called (already succeeded)
    expect(calls.cafef).toBe(0); // Not called (already succeeded)
  });

  it('falls back to Yahoo when VNStock fails', async () => {
    const adapters = [
      createMockAdapter('VNStock', true), // VNStock fails
      createMockAdapter('Yahoo', false), // Yahoo succeeds
      createMockAdapter('CafeF', false), // CafeF not reached
    ];

    let successSource = '';
    for (const adapter of adapters) {
      const result = await adapter.fetchHistorical('TEST', 'VN', '1d', '60d');
      if (result && result.length > 0) {
        successSource = adapter.name;
        break;
      }
    }

    expect(successSource).toBe('Yahoo');
  });

  it('returns null when all adapters fail', async () => {
    const adapters = [
      createMockAdapter('VNStock', true),
      createMockAdapter('Yahoo', true),
      createMockAdapter('CafeF', true),
    ];

    let result = null;
    for (const adapter of adapters) {
      const data = await adapter.fetchHistorical('TEST', 'VN', '1d', '60d');
      if (data && data.length > 0) {
        result = data;
        break;
      }
    }

    expect(result).toBeNull();
  });

  it('respects supports() check before attempting fetch', async () => {
    const fetchedFromUnsupported: string[] = [];

    const adapters = [
      {
        name: 'VNStock-IndicesOnly',
        supports: (symbol: string) => symbol.startsWith('^'),
        fetchStock: async () => null,
        fetchHistorical: async (symbol: string) => {
          fetchedFromUnsupported.push(symbol);
          return null;
        },
      } as DataSourceAdapter,
      {
        name: 'Yahoo',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => [{ timestamp: Math.floor(Date.now() / 1000), close: 100 }],
      } as DataSourceAdapter,
    ];

    // Fetch for a stock (not index)
    for (const adapter of adapters) {
      if (!adapter.supports('VCB', 'VN')) continue; // Skip if doesn't support
      const result = await adapter.fetchHistorical('VCB', 'VN', '1d', '60d');
      if (result && result.length > 0) break;
    }

    // VNStock-IndicesOnly should never fetch because supports() returned false
    expect(fetchedFromUnsupported).toEqual([]);
  });
});
