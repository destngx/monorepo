export const CACHE_PREFIX = 'market-pulse:';
export const PRICE_CACHE_TTL = 300; // 5 minutes during trading
export const HISTORY_CACHE_TTL = 3600; // 1 hour for historical data
export const ASSET_DATA_CACHE_TTL = 14 * 24 * 3600; // 14 days for processed asset data

export const US_TICKERS = [
  { symbol: '^VIX', name: 'VIX' },
  { symbol: 'DX-Y.NYB', name: 'DXY' },
  { symbol: '^TNX', name: 'US10Y' },
  { symbol: 'CL=F', name: 'WTI' },
  { symbol: 'GC=F', name: 'Gold' },
  { symbol: '^GSPC', name: 'S&P500' },
  { symbol: '^NDX', name: 'NQ100' },
  { symbol: 'BTC-USD', name: 'BTC' },
];

export const VN_TICKERS = [
  // Indices
  { symbol: '^VNINDEX', name: 'VN-Index' },
  { symbol: '^HNX', name: 'HNX' },
  { symbol: 'VN30', name: 'VN30' },
  { symbol: '^UPCOM', name: 'UPCOM' },
  // Major Vietnamese stocks (most liquid)
  { symbol: 'VCB', name: 'Vietcombank' },
  { symbol: 'VNM', name: 'Vinamilk' },
  { symbol: 'IFC', name: 'Imexpharm' },
  { symbol: 'ACB', name: 'ACB Bank' },
  { symbol: 'TCB', name: 'Techcombank' },
  { symbol: 'MBB', name: 'MB Bank' },
  { symbol: 'FPT', name: 'FPT Software' },
  { symbol: 'HPG', name: 'Hoa Phat' },
  { symbol: 'SAB', name: 'Sabeco' },
  { symbol: 'GAS', name: 'PV Gas' },
  // Currency
  { symbol: 'VND=X', name: 'USD/VND' },
];
