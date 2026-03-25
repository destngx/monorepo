import { describe, it, expect, vi } from 'vitest';
import type { DataSourceAdapter, StockDataPoint } from '../../data-sources';

describe('Multi-Source Fallback Chain', () => {
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

  it('tries adapters in sequence until success', async () => {
    const adapters = [
      createMockAdapter('First', true), // Fails
      createMockAdapter('Second', true), // Fails
      createMockAdapter('Third', false), // Succeeds
    ];

    let successSource = '';
    for (const adapter of adapters) {
      const result = await adapter.fetchHistorical('TEST', 'VN', '1d', '60d');
      if (result && result.length > 0) {
        successSource = adapter.name;
        break;
      }
    }

    expect(successSource).toBe('Third');
  });

  it('stops after first success (no unnecessary requests)', async () => {
    const calls = { first: 0, second: 0, third: 0 };

    const adapters = [
      {
        name: 'First',
        supports: () => false,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.first++;
          return null;
        },
      } as DataSourceAdapter,
      {
        name: 'Second',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.second++;
          return [{ timestamp: Math.floor(Date.now() / 1000), close: 100 }];
        },
      } as DataSourceAdapter,
      {
        name: 'Third',
        supports: () => true,
        fetchStock: async () => null,
        fetchHistorical: async () => {
          calls.third++;
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

    expect(calls.first).toBe(0); // Not called (doesn't support)
    expect(calls.second).toBe(1); // Called once, succeeded
    expect(calls.third).toBe(0); // Not called (already succeeded)
  });

  it('returns null when all adapters fail', async () => {
    const adapters = [
      createMockAdapter('First', true),
      createMockAdapter('Second', true),
      createMockAdapter('Third', true),
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
    const fetchedFromUnsupported = [];

    const adapters = [
      {
        name: 'Indices-Only',
        supports: (symbol: string) => symbol.startsWith('^'),
        fetchStock: async () => null,
        fetchHistorical: async (symbol: string) => {
          fetchedFromUnsupported.push(symbol);
          return null;
        },
      } as DataSourceAdapter,
      {
        name: 'AllSupport',
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

    // Indices-Only should never fetch because supports() returned false
    expect(fetchedFromUnsupported).toEqual([]);
  });
});
