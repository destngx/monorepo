import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getStockAnalysis } from './stock-analysis-service';
import * as utils from '@wealth-management/utils';
import { VNStockAdapter } from '../data-sources/vnstock';

// Mock utils
vi.mock('@wealth-management/utils', () => ({
  getCached: vi.fn(),
  setCache: vi.fn(),
}));

// Mock VNStockAdapter to avoid real network/server calls
vi.mock('../data-sources/vnstock', () => {
  return {
    VNStockAdapter: vi.fn().mockImplementation(() => ({
      fetchStock: vi.fn().mockResolvedValue({
        symbol: 'VCB',
        name: 'Vietcombank',
        lastPrice: 90000,
      }),
      fetchHistorical: vi.fn().mockResolvedValue(
        Array.from({ length: 60 }, (_, i) => ({
          timestamp: Date.now() / 1000 - i * 86400,
          close: 80000 + i * 200, // Trending up for simplicity
        })),
      ),
    })),
  };
});

describe('StockAnalysisService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should return cached analysis if available', async () => {
    const mockAnalysis = { symbol: 'VCB', name: 'Vietcombank', price: 90000 };
    vi.mocked(utils.getCached).mockResolvedValueOnce(mockAnalysis);

    const result = await getStockAnalysis('VCB');

    expect(result).toEqual(mockAnalysis);
    expect(utils.getCached).toHaveBeenCalledWith('stock-analysis:VCB');
  });

  it('should perform analysis if not in cache', async () => {
    vi.mocked(utils.getCached).mockResolvedValueOnce(null);

    const result = await getStockAnalysis('VCB');

    expect(result).not.toBeNull();
    expect(result?.symbol).toBe('VCB');
    expect(result?.name).toBe('Vietcombank');
    expect(result?.price).toBe(90000);
    expect(result?.technicals.trend).toBeDefined();
    expect(utils.setCache).toHaveBeenCalled();
  });

  it('should return null if metadata is missing', async () => {
    vi.mocked(utils.getCached).mockResolvedValueOnce(null);

    // Override manual mock for this test
    const adapter = new VNStockAdapter();
    vi.spyOn(adapter, 'fetchStock').mockResolvedValueOnce(null);

    // We need to ensure the service uses this mocked adapter instance
    // Since the service instantiates it internally, we rely on the class mock above.
    // For this specific test, we'll re-mock the constructor.
    (VNStockAdapter as any).mockImplementationOnce(() => ({
      fetchStock: vi.fn().mockResolvedValue(null),
      fetchHistorical: vi.fn().mockResolvedValue([]),
    }));

    const result = await getStockAnalysis('MISSING');
    expect(result).toBeNull();
  });
});
