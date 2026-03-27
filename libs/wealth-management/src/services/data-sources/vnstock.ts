import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';
import { getCached, setCache } from '@wealth-management/utils';

const VNSTOCK_CACHE_PREFIX = 'vnstock:';
const METADATA_CACHE_TTL = 30 * 24 * 3600; // 30 days for company metadata
const HISTORICAL_CACHE_TTL = 7 * 24 * 3600; // 7 days for historical price data
const HEALTH_CHECK_TTL = 60; // 1 minute health check cache

/**
 * Symbol mapping: internal market-data-service symbols → vnstock-server API symbols
 */
const INDEX_SYMBOL_MAP: Record<string, string> = {
  '^VNINDEX': 'VNINDEX',
  VN30: 'VN30',
};

/**
 * Check if a symbol is a VN index (requires /index-history endpoint)
 */
function isVNIndex(symbol: string): boolean {
  return symbol in INDEX_SYMBOL_MAP;
}

/**
 * Convert internal symbol to vnstock-server API symbol
 */
function toVnstockSymbol(symbol: string): string {
  return INDEX_SYMBOL_MAP[symbol] || symbol;
}

/**
 * VNStock Server Adapter
 *
 * Primary data source for all Vietnamese market data (indices + individual stocks).
 * Connects to the local vnstock-server FastAPI service.
 * Falls back gracefully (returns null) if server is unavailable.
 */
export class VNStockAdapter implements DataSourceAdapter {
  name = 'VNStock';
  private pythonServerUrl = process.env.VNSTOCK_SERVER_URL || 'http://localhost:8000';

  /**
   * Supports all VN market symbols: indices (^VNINDEX, ^HNX, VN30, ^UPCOM) and individual stocks (VCB, FPT, etc.)
   */
  supports(symbol: string, market: 'US' | 'VN'): boolean {
    return market === 'VN';
  }

  /**
   * Health check: ping vnstock-server with cached result to avoid repeated failures
   */
  async isServerAvailable(): Promise<boolean> {
    const cacheKey = `${VNSTOCK_CACHE_PREFIX}health`;
    const cached = await getCached<boolean>(cacheKey);
    if (cached !== null && cached !== undefined) return cached;

    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 3000); // 3s timeout

      const res = await fetch(`${this.pythonServerUrl}/health`, {
        signal: controller.signal,
      });
      clearTimeout(timeout);

      const available = res.ok;
      await setCache(cacheKey, available, HEALTH_CHECK_TTL);
      return available;
    } catch {
      await setCache(cacheKey, false, HEALTH_CHECK_TTL);
      return false;
    }
  }

  async fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null> {
    if (!this.supports(symbol, market)) return null;

    const cacheKey = `${VNSTOCK_CACHE_PREFIX}stock:${symbol}`;

    // Check cache first
    const cached = await getCached<StockMetadata>(cacheKey);
    if (cached) {
      console.log(`[VNStockAdapter] ✓ Cache hit for stock metadata: ${symbol}`);
      return cached;
    }

    // Health check before attempting request
    if (!(await this.isServerAvailable())) {
      console.log(`[VNStockAdapter] Server unavailable, skipping fetchStock for ${symbol}`);
      return null;
    }

    try {
      const apiSymbol = toVnstockSymbol(symbol);
      const res = await fetch(`${this.pythonServerUrl}/api/v1/stocks/quote?symbol=${encodeURIComponent(apiSymbol)}`);
      if (!res.ok) return null;

      const data = await res.json();
      if (!data.success) return null;

      const metadata: StockMetadata = {
        symbol,
        name: data.data.name || symbol,
        market: 'VN',
        currency: 'VND',
        lastPrice: data.data.price,
        lastUpdate: Date.now(),
      };

      // Cache metadata for 30 days
      await setCache(cacheKey, metadata, METADATA_CACHE_TTL);
      console.log(`[VNStockAdapter] Cached stock metadata for 30 days: ${symbol}`);

      return metadata;
    } catch (error) {
      const networkError = isAppError(error)
        ? error
        : new NetworkError(`VNStock: Failed to fetch stock ${symbol}`, {
            context: { symbol, market, source: 'vnstock' },
          });
      console.error('[VNStockAdapter]', networkError.message);
      return null;
    }
  }

  async fetchHistorical(
    symbol: string,
    market: 'US' | 'VN',
    interval: string,
    range: string,
  ): Promise<StockDataPoint[] | null> {
    if (!this.supports(symbol, market)) return null;

    const cacheKey = `${VNSTOCK_CACHE_PREFIX}historical:${symbol}:${interval}:${range}`;

    // Check cache first
    const cached = await getCached<StockDataPoint[]>(cacheKey);
    if (cached) {
      console.log(`[VNStockAdapter] ✓ Cache hit for historical data: ${symbol} (${range})`);
      return cached;
    }

    // Health check before attempting request
    if (!(await this.isServerAvailable())) {
      console.log(`[VNStockAdapter] Server unavailable, skipping fetchHistorical for ${symbol}`);
      return null;
    }

    try {
      const days = parseInt(range.replace(/[^\d]/g, '')) || 30;
      const start = new Date();
      start.setDate(start.getDate() - days);
      const startDate = start.toISOString().split('T')[0];

      const apiSymbol = toVnstockSymbol(symbol);
      let fetchUrl: string;

      if (isVNIndex(symbol)) {
        // Use dedicated index-history endpoint for indices
        fetchUrl = `${this.pythonServerUrl}/api/v1/stocks/index-history?symbol=${encodeURIComponent(apiSymbol)}&start_date=${startDate}`;
      } else {
        // Use standard historical endpoint for individual stocks
        let resolution = interval.toUpperCase();
        if (resolution === '1H') resolution = '1H';
        fetchUrl = `${this.pythonServerUrl}/api/v1/stocks/historical?symbol=${encodeURIComponent(apiSymbol)}&resolution=${resolution}&start_date=${startDate}`;
      }

      const realRes = await fetch(fetchUrl);
      if (!realRes.ok) return null;

      const data = await realRes.json();
      if (!data.success || !data.data) return null;

      const historicalData = (data.data as any[]).map((candle: any) => ({
        timestamp: Math.floor(new Date(candle.time || candle.date).getTime() / 1000),
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
        volume: candle.volume,
      }));

      // Cache historical data for 7 days
      await setCache(cacheKey, historicalData, HISTORICAL_CACHE_TTL);
      console.log(`[VNStockAdapter] Cached historical data for 7 days: ${symbol} (${range})`);

      return historicalData;
    } catch (error) {
      const networkError = isAppError(error)
        ? error
        : new NetworkError(`VNStock: Failed to fetch historical data for ${symbol}`, {
            context: { symbol, market, interval, range, source: 'vnstock' },
          });
      console.error('[VNStockAdapter]', networkError.message);
      return null;
    }
  }
}
