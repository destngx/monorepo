import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';

export class VNStockAdapter implements DataSourceAdapter {
  name = 'VNStock';
  private pythonServerUrl = process.env.VNSTOCK_SERVER_URL || 'http://localhost:3001';

  supports(symbol: string, market: 'US' | 'VN'): boolean {
    // VNSTOCK only supports Vietnamese market
    return market === 'VN';
  }

  async fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null> {
    if (!this.supports(symbol, market)) return null;

    try {
      const res = await fetch(`${this.pythonServerUrl}/api/vnstock/quote?symbol=${encodeURIComponent(symbol)}`);
      if (!res.ok) return null;

      const data = await res.json();
      if (!data.success) return null;

      return {
        symbol,
        name: data.data.name || symbol,
        market: 'VN',
        currency: 'VND',
        lastPrice: data.data.price,
        lastUpdate: Date.now(),
      };
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

    try {
      const res = await fetch(
        `${this.pythonServerUrl}/api/vnstock/history?symbol=${encodeURIComponent(symbol)}&interval=${interval}&range=${range}`,
      );
      if (!res.ok) return null;

      const data = await res.json();
      if (!data.success || !data.data) return null;

      return (data.data as any[]).map((candle: any) => ({
        timestamp: Math.floor(new Date(candle.date).getTime() / 1000),
        open: candle.open,
        high: candle.high,
        low: candle.low,
        close: candle.close,
        volume: candle.volume,
      }));
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
