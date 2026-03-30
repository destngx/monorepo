import { DataSourceAdapter, StockMetadata, StockDataPoint } from './types';
import { NetworkError, isAppError } from '../../utils/errors';

export class CafeFAdapter implements DataSourceAdapter {
  name = 'CafeF';

  supports(symbol: string, market: 'US' | 'VN'): boolean {
    // CafeF only supports Vietnamese indices via their undocumented endpoint
    const supportedIndices = ['^VNINDEX', '^HNX', '^UPCOM', 'VN30'];
    return market === 'VN' && supportedIndices.some((idx) => symbol.includes(idx));
  }

  async fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null> {
    if (!this.supports(symbol, market)) return null;

    try {
      const indexMap: Record<string, number> = {
        '^VNINDEX': 1,
        '^HNX': 2,
        VN30: 11,
        '^UPCOM': 9,
      };

      const indexId = Object.entries(indexMap).find(([key]) => symbol.includes(key))?.[1];
      if (!indexId) return null;

      const res = await fetch(
        `https://cafef.vn/du-lieu/Ajax/PageNew/RealtimeChartHeader.ashx?index=${indexId}&type=market`,
        {
          headers: {
            'User-Agent':
              'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
            Referer: 'https://cafef.vn/',
          },
        },
      );

      if (!res.ok) return null;
      const data = await res.json();

      if (!data[indexId]) return null;
      const indexData = data[indexId];

      return {
        symbol,
        name: symbol.replace(/[^\w]/g, ''),
        market: 'VN',
        currency: 'VNPoints',
        lastPrice: indexData.CurrentIndex,
        lastUpdate: Date.now(),
      };
    } catch (error) {
      const networkError = isAppError(error)
        ? error
        : new NetworkError(`CafeF: Failed to fetch index ${symbol}`, {
            context: { symbol, market, source: 'cafef' },
          });
      console.error('[CafeFAdapter]', networkError.message);
      return null;
    }
  }

  async fetchHistorical(
    symbol: string,
    market: 'US' | 'VN',
    interval: string,
    range: string,
  ): Promise<StockDataPoint[] | null> {
    // CafeF doesn't provide historical data via free endpoint
    return null;
  }
}
