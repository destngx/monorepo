import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';

export class VNStockAdapter implements DataSourceAdapter {
  name = 'VNStock';
  private pythonServerUrl = process.env.VNSTOCK_SERVER_URL || 'http://localhost:8000';

  supports(symbol: string, market: 'US' | 'VN'): boolean {
    return market === 'VN';
  }

  async fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null> {
    if (!this.supports(symbol, market)) return null;

    try {
      const res = await fetch(`${this.pythonServerUrl}/api/v1/stocks/quote?symbol=${encodeURIComponent(symbol)}`);
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
      // Map interval and range to what FastAPI expects
      // range format: "7d", "30d", "60d"
      let resolution = interval.toUpperCase();
      if (resolution === '1H') resolution = '1H'; // FastAPI supports 1H

      const res = await fetch(
        `${this.pythonServerUrl}/api/v1/stocks/historical?symbol=${encodeURIComponent(symbol)}&resolution=${resolution}&range_shortcut=${range}`,
      );

      // Update: main.py needs to support range_shortcut or we calculate it here
      // I'll calculate it here to be safer or updated main.py
      const days = parseInt(range.replace(/[^\d]/g, '')) || 30;
      const start = new Date();
      start.setDate(start.getDate() - days);
      const startDate = start.toISOString().split('T')[0];

      const fetchUrl = `${this.pythonServerUrl}/api/v1/stocks/historical?symbol=${encodeURIComponent(symbol)}&resolution=${resolution}&start_date=${startDate}`;

      const realRes = await fetch(fetchUrl);
      if (!realRes.ok) return null;

      const data = await realRes.json();
      if (!data.success || !data.data) return null;

      return (data.data as any[]).map((candle: any) => ({
        timestamp: Math.floor(new Date(candle.time || candle.date).getTime() / 1000),
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
