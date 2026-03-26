import { getCached, setCache } from '@wealth-management/utils';
import { NetworkError, isAppError } from '@wealth-management/utils';
import { DataSourceAdapter, YahooFinanceAdapter, VNStockAdapter, CafeFAdapter } from '../../data-sources';
import { MarketAsset, MarketState } from './types';
import { generateRealCorrelationMatrix, generateTechnicals } from './technical-analysis';
import { generateValuation } from './valuation';
import { computeCapitalFlow } from './ai-analysis';
import { ASSET_DATA_CACHE_TTL, CACHE_PREFIX } from './constants';

const dataSourceAdapters = [
  new CafeFAdapter(), // Indices only
  new VNStockAdapter(), // Vietnamese stocks
  new YahooFinanceAdapter(), // Fallback
];

/**
 * Maps timeframe to data source-specific intervals and ranges
 */
function getIntervalAndRange(
  timeframe: string,
  sourceType: 'vnstock' | 'yahoo' | 'cafef',
): { interval: string; range: string } {
  switch (timeframe) {
    case '1h':
      return { interval: '1h', range: '7d' };
    case '4h':
      return sourceType === 'vnstock' ? { interval: '4h', range: '14d' } : { interval: '1h', range: '14d' };
    case '1d':
      return { interval: '1d', range: '60d' };
    case '1w':
      return { interval: '1w', range: '90d' };
    default:
      return { interval: '1h', range: '7d' };
  }
}

/**
 * Fetches asset data using multi-source fallback chain
 */
export async function fetchAssetData(
  symbol: string,
  name: string,
  market: 'US' | 'VN',
  timeframe = '1h',
): Promise<MarketAsset | null> {
  const cacheKey = `${CACHE_PREFIX}asset:${symbol}:${market}:${timeframe}`;

  // Check cache first
  const cached = await getCached<MarketAsset>(cacheKey);
  if (cached) {
    return cached;
  }

  try {
    // Try each adapter in sequence
    for (const adapter of dataSourceAdapters) {
      if (!adapter.supports(symbol, market)) continue;

      const sourceType = adapter.name.toLowerCase().replace(/[^a-z0-9]/g, '');
      const { interval, range } = getIntervalAndRange(timeframe, sourceType as 'vnstock' | 'yahoo' | 'cafef');

      const dataPoints = await adapter.fetchHistorical(symbol, market, interval, range);
      if (!dataPoints || dataPoints.length < 2) continue;

      const closes = dataPoints.map((p) => p.close);
      const highs = dataPoints.map((p) => p.high || p.close);
      const lows = dataPoints.map((p) => p.low || p.close);
      const timestamps = dataPoints.map((p) => p.timestamp);

      const currentPrice = closes[closes.length - 1];

      let offset = 2;
      if (timeframe === '4h') offset = 5;
      if (timeframe === '1w') offset = 6;

      const prevPriceIdx = Math.max(0, closes.length - offset);
      const prevPrice = closes[prevPriceIdx] || currentPrice;
      const percentChange = prevPrice ? ((currentPrice - prevPrice) / prevPrice) * 100 : 0;

      let oneDayAgoIdx = 0;
      let oneWeekAgoIdx = 0;

      if (interval === '1h') {
        oneDayAgoIdx = Math.max(0, closes.length - 8);
        oneWeekAgoIdx = Math.max(0, closes.length - 40);
      } else if (interval === '1d') {
        oneDayAgoIdx = Math.max(0, closes.length - 2);
        oneWeekAgoIdx = Math.max(0, closes.length - 6);
      }

      const dayChange = closes[oneDayAgoIdx] ? ((currentPrice - closes[oneDayAgoIdx]) / closes[oneDayAgoIdx]) * 100 : 0;
      const weekChange = closes[oneWeekAgoIdx]
        ? ((currentPrice - closes[oneWeekAgoIdx]) / closes[oneWeekAgoIdx]) * 100
        : 0;

      const direction = percentChange > 0.05 ? 'up' : percentChange < -0.05 ? 'down' : 'flat';

      let momentum: 'fire' | 'stable' | 'sleep' = 'stable';
      if (Math.abs(percentChange) > 2 || (direction === 'up' && dayChange > 5)) momentum = 'fire';
      else if (Math.abs(percentChange) < 0.1) momentum = 'sleep';

      const assetData: MarketAsset = {
        symbol: name,
        name,
        market,
        price: currentPrice,
        percentChange,
        dayChange,
        weekChange,
        direction,
        momentum,
        closes,
        highs,
        lows,
        timestamps,
      };

      await setCache(cacheKey, assetData, ASSET_DATA_CACHE_TTL);
      return assetData;
    }

    return null;
  } catch (error) {
    return null;
  }
}

async function fetchVNIndicesFromCafeF(): Promise<any | null> {
  try {
    const url = `https://cafef.vn/du-lieu/Ajax/PageNew/RealtimeChartHeader.ashx?index=1;2;9;11;12&type=market`;
    const res = await fetch(url, {
      headers: {
        'User-Agent':
          'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36',
        Referer: 'https://cafef.vn/',
      },
    });
    if (!res.ok) return null;
    return await res.json();
  } catch (e) {
    return null;
  }
}

async function fetchUSDVNDFromCurrencyAPI(): Promise<MarketAsset | null> {
  try {
    const res = await fetch('https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.json');
    if (!res.ok) return null;
    const data = await res.json();
    const price = data.usd.vnd;
    return {
      symbol: 'USD/VND',
      name: 'USD/VND',
      market: 'VN',
      price,
      percentChange: 0.1,
      dayChange: 0.1,
      weekChange: 0.2,
      direction: 'up',
      momentum: 'stable',
    };
  } catch (e) {
    return null;
  }
}

/**
 * Fetches data for a group of tickers and computes state
 */
export async function fetchMarketGroup(
  tickers: { symbol: string; name: string }[],
  market: 'US' | 'VN',
  timeframe: string,
): Promise<MarketState> {
  let validAssets: MarketAsset[] = [];

  if (market === 'VN' && timeframe === '1h') {
    const [vnIndices, fxData] = await Promise.all([fetchVNIndicesFromCafeF(), fetchUSDVNDFromCurrencyAPI()]);

    const assetPromises = tickers.map((t) => fetchAssetData(t.symbol, t.name, market, timeframe));
    const historicalAssets = await Promise.all(assetPromises);

    const indexMap: Record<string, number> = {
      'VN-Index': 1,
      HNX: 2,
      VN30: 11,
      UPCOM: 9,
    };

    tickers.forEach((t) => {
      const histAsset = historicalAssets.find((h) => h && h.name === t.name);
      const closes = histAsset?.closes || [];

      if (t.symbol === 'VND=X' && fxData) {
        fxData.closes = closes;
        validAssets.push(fxData);
      } else {
        const cafeId = indexMap[t.name];
        const data = vnIndices ? vnIndices[cafeId] : null;

        if (data) {
          const current = data.CurrentIndex;
          const prev = data.PrevIndex;
          const diff = current - prev;
          const percent = (diff / prev) * 100;

          if (closes.length && Math.abs(closes[closes.length - 1] - current) > 0.01) {
            closes.push(current);
          }

          validAssets.push({
            symbol: t.name,
            name: t.name,
            market: 'VN',
            price: current,
            percentChange: percent,
            dayChange: percent,
            weekChange: histAsset ? histAsset.weekChange : percent,
            direction: percent > 0.05 ? 'up' : percent < -0.05 ? 'down' : 'flat',
            momentum: Math.abs(percent) > 1.5 ? 'fire' : 'stable',
            closes,
          });
        } else if (histAsset) {
          validAssets.push(histAsset);
        }
      }
    });
  } else {
    const assetPromises = tickers.map((t) => fetchAssetData(t.symbol, t.name, market, timeframe));
    const assets = await Promise.all(assetPromises);
    validAssets = assets.filter((a): a is MarketAsset => a !== null);
  }

  const sortedByChange = [...validAssets].sort((a, b) => Math.abs(b.percentChange) - Math.abs(a.percentChange));
  const topMovers = sortedByChange.slice(0, 3).map((a) => ({ symbol: a.name, change: a.percentChange }));
  const moversStr = topMovers.map((m) => `${m.symbol} (${m.change > 0 ? '+' : ''}${m.change.toFixed(2)}%)`).join(', ');

  const capitalFlow = computeCapitalFlow(validAssets, market);
  const assetList = validAssets.map((a) => a.name);
  const correlationMatrix = generateRealCorrelationMatrix(validAssets);

  return {
    assets: validAssets,
    drivers: {
      topMovers,
      summaryEn: `Top movers: ${moversStr}.`,
      summaryVi: `Các tài sản biến động mạnh nhất: ${moversStr}.`,
    },
    capitalFlow,
    correlationMatrix,
    assetList,
    technicals: generateTechnicals(validAssets, market),
    valuation: generateValuation(validAssets, market),
  };
}
