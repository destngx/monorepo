import { describe, it, expect, vi, beforeEach } from 'vitest';
import { getExchangeRate } from './exchange-rate-service';
import * as cache from '../db/cache';

// Mock the cache module
vi.mock('../db/cache', () => ({
  getCached: vi.fn(),
  setCache: vi.fn()
}));

// Mock global fetch
const globalFetch = vi.fn();
global.fetch = globalFetch;

describe('exchange-rate-service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should return cached rate if available', async () => {
    vi.mocked(cache.getCached).mockResolvedValueOnce({ rate: 26000 });

    const rate = await getExchangeRate();

    expect(rate).toBe(26000);
    expect(globalFetch).not.toHaveBeenCalled();
    expect(cache.setCache).not.toHaveBeenCalled();
  });

  it('should fetch from CoinGecko and cache if not in cache', async () => {
    vi.mocked(cache.getCached).mockResolvedValueOnce(null);
    globalFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ tether: { vnd: 25500 } })
    });

    const rate = await getExchangeRate();

    expect(rate).toBe(25500);
    expect(globalFetch).toHaveBeenCalledTimes(1);
    expect(cache.setCache).toHaveBeenCalledWith('exchange-rate:usdt-vnd', { rate: 25500 }, 900);
  });

  it('should use fallback rate if fetch fails', async () => {
    vi.mocked(cache.getCached).mockResolvedValueOnce(null);
    globalFetch.mockRejectedValueOnce(new Error('Network error'));

    // Suppress console.warn for the test
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => { });

    const rate = await getExchangeRate();

    expect(rate).toBe(25400); // 25400 is the hardcoded fallback
    expect(consoleSpy).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });
});
