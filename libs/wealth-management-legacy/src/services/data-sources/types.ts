/**
 * Data Source Adapter Types
 * Defines the contract for all data source implementations
 */

export interface StockDataPoint {
  timestamp: number; // Unix timestamp (seconds)
  open?: number;
  high?: number;
  low?: number;
  close: number;
  volume?: number;
}

export interface StockMetadata {
  symbol: string;
  name: string;
  market: 'US' | 'VN';
  currency?: string;
  lastPrice: number;
  lastUpdate: number; // Unix timestamp
}

export interface DataSourceResult {
  success: boolean;
  data?: {
    metadata: StockMetadata;
    candles: StockDataPoint[];
  };
  error?: string;
  source: string;
}

export interface DataSourceAdapter {
  /**
   * Fetch current stock price and metadata
   */
  fetchStock(symbol: string, market: 'US' | 'VN'): Promise<StockMetadata | null>;

  /**
   * Fetch historical OHLCV data
   * @param symbol Stock ticker
   * @param market Market (US or VN)
   * @param interval Candle interval (1h, 1d, 1w)
   * @param range Time range (7d, 60d, 90d, 1y)
   */
  fetchHistorical(
    symbol: string,
    market: 'US' | 'VN',
    interval: string,
    range: string,
  ): Promise<StockDataPoint[] | null>;

  /**
   * Name of this data source for logging/debugging
   */
  name: string;

  /**
   * Whether this source supports the given symbol/market combination
   */
  supports(symbol: string, market: 'US' | 'VN'): boolean;
}
