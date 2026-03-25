import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';

export class YahooFinanceAdapter implements DataSourceAdapter {
  name = 'YahooFinance';

  supports(symbol: string, market: 'US' | 'VN'): boolean {
    // Yahoo supports both, but VN support is unreliable
    return true;
  }

  async fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null> {
    try {
      const url = `https://query1.finance.yahoo.com/v8/finance/chart/${encodeURIComponent(symbol)}?range=1d&interval=1d`;
      const res = await fetch(url);
      if (!res.ok) return null;

      const data = await res.json();
      const result = data.chart.result?.[0];
      if (!result) return null;

      const meta = result.meta;
      const price = meta.regularMarketPrice;

      return {
        symbol,
        name: meta.longName || symbol,
        market,
        currency: meta.currency,
        lastPrice: price,
        lastUpdate: meta.regularMarketTime * 1000,
      };
    } catch (error) {
      const networkError = isAppError(error)
        ? error
        : new NetworkError(`Yahoo Finance: Failed to fetch stock ${symbol}`, {
            context: { symbol, market, source: 'yahoo-finance' },
          });
      console.error('[YahooFinanceAdapter]', networkError.message);
      return null;
    }
  }

  async fetchHistorical(
    symbol: string,
    market: 'US' | 'VN',
    interval: string,
    range: string,
  ): Promise<StockDataPoint[] | null> {
    try {
      const url = `https://query1.finance.yahoo.com/v8/finance/chart/${encodeURIComponent(symbol)}?range=${range}&interval=${interval}`;
      const res = await fetch(url);
      if (!res.ok) return null;

      const data = await res.json();
      const result = data.chart.result?.[0];
      if (!result) return null;

      const quotes = result.indicators.quote[0];
      if (!quotes || !quotes.close) return null;

      const timestamps = result.timestamp || [];
      const closesRaw = quotes.close || [];
      const highsRaw = quotes.high || [];
      const lowsRaw = quotes.low || [];
      const opensRaw = quotes.open || [];
      const volumesRaw = quotes.volume || [];

      const validPairs = timestamps
        .map((t: number, i: number) => ({
          timestamp: t,
          open: opensRaw[i],
          high: highsRaw[i],
          low: lowsRaw[i],
          close: closesRaw[i],
          volume: volumesRaw[i],
        }))
        .filter((p: any) => p.close !== null && p.close !== undefined);

      return validPairs;
    } catch (error) {
      const networkError = isAppError(error)
        ? error
        : new NetworkError(`Yahoo Finance: Failed to fetch historical data for ${symbol}`, {
            context: { symbol, market, interval, range, source: 'yahoo-finance' },
          });
      console.error('[YahooFinanceAdapter]', networkError.message);
      return null;
    }
  }
}
