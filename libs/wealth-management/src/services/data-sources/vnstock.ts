import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';
import { getCached, setCache } from '@wealth-management/utils';

const VNSTOCK_CACHE_PREFIX = 'vnstock:';
const METADATA_CACHE_TTL = 30 * 24 * 3600; // 30 days for company metadata
const HISTORICAL_CACHE_TTL = 7 * 24 * 3600; // 7 days for historical price data

export class VNStockAdapter implements DataSourceAdapter {
  name = 'VNStock';
  private pythonServerUrl = process.env.VNSTOCK_SERVER_URL || 'http://localhost:8000';

  supports(symbol: string, market: 'US' | 'VN'): boolean {
    return market === 'VN';
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

    try {
      const res = await fetch(`${this.pythonServerUrl}/api/v1/stocks/quote?symbol=${encodeURIComponent(symbol)}`);
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

    try {
      const days = parseInt(range.replace(/[^\d]/g, '')) || 30;
      const start = new Date();
      start.setDate(start.getDate() - days);
      const startDate = start.toISOString().split('T')[0];

      let resolution = interval.toUpperCase();
      if (resolution === '1H') resolution = '1H';

      const fetchUrl = `${this.pythonServerUrl}/api/v1/stocks/historical?symbol=${encodeURIComponent(symbol)}&resolution=${resolution}&start_date=${startDate}`;

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
