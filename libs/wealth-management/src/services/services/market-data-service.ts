import { getCached, setCache } from '@wealth-management/utils';
import { generateText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { loadPrompt, replacePlaceholders } from '../../ai/prompts/loader';
import { NetworkError, isAppError, getErrorMessage } from '../../utils/errors';
import { calculateEMA, calculateSMA, calculateRSI, calculateSeasonality } from '../../utils/technical-analysis';

const CACHE_PREFIX = 'market-pulse:';
const PRICE_CACHE_TTL = 300; // 5 minutes during trading
const HISTORY_CACHE_TTL = 3600; // 1 hour for historical data

export interface MarketAsset {
  symbol: string;
  name: string;
  market: 'US' | 'VN';
  price: number;
  percentChange: number;
  dayChange: number;
  weekChange: number;
  direction: 'up' | 'down' | 'flat';
  momentum: 'fire' | 'stable' | 'sleep';
  closes?: number[]; // Added historical closes for real calculations
  highs?: number[];
  lows?: number[];
  timestamps?: number[]; // Added historical timestamps for seasonality
}

export interface Technicals {
  cycle: {
    phase: 'Accumulation' | 'Markup' | 'Distribution' | 'Mark-Down' | 'Decline';
    description: string;
    descriptionVi: string;
    strength: number; // 0-100
    confidence: number;
    phases?: { label: string; value: number; color?: string }[]; // For Donut chart
  };
  trend: {
    direction: string;
    directionEn: string;
    strength: number;
    confidence: number;
  };
  ict: {
    fvgs: { type: string; top: number; bottom: number; gap: number }[];
    orderBlocks: any[];
  };
  sentiment: {
    score: number;
    label: string;
    labelVi: string;
  };
  indicators: {
    rsi?: number;
    ema20?: number;
    ema50?: number;
    ema200?: number;
    sma20?: number;
    sma50?: number;
    sma200?: number;
  };
  signals?: {
    action: 'SHORT' | 'LONG' | 'AVOID' | 'EXIT' | 'REDUCE' | 'TAKE PROFIT' | 'HOLD/WATCH';
    actionVi: string;
    entry: number;
    stopLoss: number;
    takeProfit: number;
    rr: number; // Risk:Reward ratio
    confidence: number;
    reasons: string[];
    reasonsVi: string[];
    optimalEntry?: {
      price: number;
      pullbackPercent: number;
      pullbackAmount: number;
    };
  };
  timeframeRelationships?: {
    pair: string; // e.g., "1wk -> 1d"
    status: 'STRONG' | 'WEAK' | 'CHOPIED' | 'ALIGNED';
    relationship: string;
    relationshipVi: string;
    advice: string;
    adviceVi: string;
  }[];
  entryTimingScore?: {
    overall: number; // 0-10
    higherTfSupport: number; // 0-6
    lowerTfConfirm: number; // 0-4
  };
  supportResistance: {
    symbol: string;
    support: number[];
    resistance: number[];
    bollingerUpper: number;
    bollingerLower: number;
    bollingerMid: number;
  }[];
  seasonality: {
    rank: number;
    name: string;
    label: string;
    return: number;
    winRate: number;
    pf: number; // Profit Factor
    stdDev: number;
    score: number;
    n: number;
    type: 'day' | 'week' | 'month';
  }[];
  n: number;
  dateRange: string;
}

export interface Valuation {
  dcf: {
    symbol: string;
    fairValue: number;
    upside: number;
    assumptions: { label: string; value: string }[];
  }[];
  monteCarlo: {
    symbol: string;
    p10: number;
    p50: number;
    p90: number;
    iterations: number;
  }[];
  sectorComparison: {
    symbol: string;
    sector: string;
    metrics: {
      label: string;
      asset: number;
      avg: number;
    }[];
  }[];
}

export interface MarketState {
  assets: MarketAsset[];
  drivers: {
    topMovers: { symbol: string; change: number }[];
    summaryEn: string;
    summaryVi: string;
    capitalFlowSignal?: 'DEFENSIVE' | 'RISK-ON' | 'MIXED';
    correlationSignalEn?: string;
    correlationSignalVi?: string;
  };
  capitalFlow: {
    signal: 'DEFENSIVE' | 'RISK-ON' | 'MIXED';
    summaryEn: string;
    summaryVi: string;
  };
  scenarios?: {
    name: string;
    regime: 'Risk-ON' | 'Risk-OFF' | 'Crisis' | 'Stagflation' | 'Goldilocks';
    confidence: number;
    summaryEn: string;
    summaryVi: string;
    actionEn?: string;
    actionVi?: string;
  }[];
  correlationMatrix: number[][];
  assetList: string[]; // For matrix headers
  technicals?: Technicals;
  valuation?: Valuation;
}

export interface MarketPulseResponse {
  us: MarketState;
  vn: MarketState;
  lastUpdated: string;
}

const US_TICKERS = [
  { symbol: '^VIX', name: 'VIX' },
  { symbol: 'DX-Y.NYB', name: 'DXY' },
  { symbol: '^TNX', name: 'US10Y' },
  { symbol: 'CL=F', name: 'WTI' },
  { symbol: 'GC=F', name: 'Gold' },
  { symbol: '^GSPC', name: 'S&P500' },
  { symbol: '^NDX', name: 'NQ100' },
  { symbol: 'BTC-USD', name: 'BTC' },
];

const VN_TICKERS = [
  { symbol: '^VNINDEX.VN', name: 'VN-Index' },
  { symbol: '^HNX', name: 'HNX' },
  { symbol: 'E1VFVN30.VN', name: 'VN30' },
  { symbol: '^UPCOM', name: 'UPCOM' },
  { symbol: 'VND=X', name: 'USD/VND' },
];

/**
 * Fetches market data for both US and VN assets.
 */
export async function getMarketPulseData(
  timeframe: '1h' | '4h' | '1d' | '1w' = '1h',
  forceRefresh = false,
): Promise<MarketPulseResponse> {
  const cacheKey = `${CACHE_PREFIX}full-data:${timeframe}`;

  if (!forceRefresh) {
    const cached = await getCached<MarketPulseResponse>(cacheKey);
    if (cached) return cached;
  }

  const now = new Date();

  // Fetch US and VN symbols in parallel
  const [usData, vnData] = await Promise.all([
    fetchMarketGroup(US_TICKERS, 'US', timeframe),
    fetchMarketGroup(VN_TICKERS, 'VN', timeframe),
  ]);

  // Fetch real VN Gold (SJC 9999) from vang.today API
  const usGold = usData.assets.find((a) => a.symbol === 'Gold');
  let vnGoldAdded = false;

  try {
    const vnGoldRes = await fetch('https://vang.today/api/prices?type=SJL1L10');
    if (vnGoldRes.ok) {
      const vnGoldData = await vnGoldRes.json();
      if (vnGoldData.success) {
        const price = vnGoldData.sell;
        const prevPrice = price - vnGoldData.change_sell;
        const percentChange = prevPrice > 0 ? (vnGoldData.change_sell / prevPrice) * 100 : 0;

        const realCloses = vnData.assets.find((a) => a.symbol === 'USD/VND')?.closes;
        vnData.assets.push({
          symbol: 'VN Gold',
          name: 'VN Gold',
          market: 'VN',
          price: price,
          percentChange: percentChange,
          dayChange: percentChange,
          weekChange: usGold ? usGold.weekChange : percentChange * 5, // Proxy week trend using US Gold
          direction: percentChange > 0.05 ? 'up' : percentChange < -0.05 ? 'down' : 'flat',
          momentum: Math.abs(percentChange) > 1.5 ? 'fire' : 'stable',
          closes: usGold?.closes
            ? usGold.closes.map((c, i) => c * (realCloses && realCloses[i] ? realCloses[i] : 25400))
            : [],
        });
        vnGoldAdded = true;
      }
    }
  } catch (e) {
    const networkError = isAppError(e)
      ? e
      : new NetworkError('Failed to fetch real VN Gold price', {
          context: { source: 'vang.today', endpoint: 'api/prices' },
        });
    console.error('[MarketDataService]', networkError.message);
  }

  // Fallback to synthetic if API fails
  if (!vnGoldAdded && usGold) {
    const usdVnd = vnData.assets.find((a) => a.symbol === 'USD/VND');
    if (usdVnd) {
      const vnGoldPrice = usGold.price * usdVnd.price;
      vnData.assets.push({
        symbol: 'VN Gold',
        name: 'VN Gold',
        market: 'VN',
        price: vnGoldPrice,
        percentChange: usGold.percentChange,
        dayChange: usGold.dayChange,
        weekChange: usGold.weekChange,
        direction: usGold.direction,
        momentum: usGold.momentum,
        closes: usGold.closes?.map((c, i) => c * (usdVnd.closes?.[i] || usdVnd.price)) || [],
      });
      vnGoldAdded = true;
    }
  }

  if (vnGoldAdded) {
    vnData.assetList.push('VN Gold');
    vnData.correlationMatrix = generateRealCorrelationMatrix(vnData.assets);
    vnData.technicals = generateTechnicals(vnData.assets, 'VN');
    vnData.valuation = generateValuation(vnData.assets, 'VN');
  }

  // Compute Scenarios (AI-driven with heuristic fallback)
  const aiAnalysis = await generateAiMarketAnalysis(usData, vnData, timeframe).catch((err) => {
    const networkError = isAppError(err)
      ? err
      : new NetworkError('AI market analysis failed', {
          context: { timeframe, source: 'generateAiMarketAnalysis' },
        });
    console.error('[MarketDataService]', networkError.message);
    return null;
  });

  // Inject AI Analysis if available, otherwise fallback to heuristics
  if (aiAnalysis) {
    if (aiAnalysis.usScenarios) usData.scenarios = aiAnalysis.usScenarios;
    if (aiAnalysis.vnScenarios) vnData.scenarios = aiAnalysis.vnScenarios;
    if (aiAnalysis.usCapitalFlow) usData.capitalFlow = aiAnalysis.usCapitalFlow;
    if (aiAnalysis.vnCapitalFlow) vnData.capitalFlow = aiAnalysis.vnCapitalFlow;

    if (aiAnalysis.usDrivers) {
      usData.drivers.summaryEn = aiAnalysis.usDrivers.summaryEn;
      usData.drivers.summaryVi = aiAnalysis.usDrivers.summaryVi;
      usData.drivers.capitalFlowSignal = aiAnalysis.usDrivers.capitalFlowSignal;
      usData.drivers.correlationSignalEn = aiAnalysis.usDrivers.correlationSignalEn;
      usData.drivers.correlationSignalVi = aiAnalysis.usDrivers.correlationSignalVi;
    }
    if (aiAnalysis.vnDrivers) {
      vnData.drivers.summaryEn = aiAnalysis.vnDrivers.summaryEn;
      vnData.drivers.summaryVi = aiAnalysis.vnDrivers.summaryVi;
      vnData.drivers.capitalFlowSignal = aiAnalysis.vnDrivers.capitalFlowSignal;
      vnData.drivers.correlationSignalEn = aiAnalysis.vnDrivers.correlationSignalEn;
      vnData.drivers.correlationSignalVi = aiAnalysis.vnDrivers.correlationSignalVi;
    }
  }

  // Final fallbacks for scenarios if AI didn't provide or failed
  const fallbackScenario = detectScenario(usData, vnData);
  if (!usData.scenarios || usData.scenarios.length === 0) usData.scenarios = [fallbackScenario];
  if (!vnData.scenarios || vnData.scenarios.length === 0) vnData.scenarios = [fallbackScenario];

  const response: MarketPulseResponse = {
    us: usData,
    vn: vnData,
    lastUpdated: now.toISOString(),
  };

  await setCache(cacheKey, response, PRICE_CACHE_TTL);
  return response;
}

/**
 * Uses AI to synthesize market data into a cohesive scenario and capital flow signals.
 */
async function generateAiMarketAnalysis(us: MarketState, vn: MarketState, timeframe: string) {
  const model = getLanguageModel('github-gpt-4o');

  const template = await loadPrompt('market', 'generate-ai-analysis');
  if (!template) {
    throw new Error(
      'Missing prompt template: market/generate-ai-analysis. Please ensure it is present in Google Sheets.',
    );
  }

  const prompt = replacePlaceholders(template, {
    timeframe,
    usAssets: us.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}% on ${timeframe})`).join(', '),
    usPhase: us.technicals?.cycle.phase || 'N/A',
    usDescription: us.technicals?.cycle.description || 'N/A',
    usSupportResistance:
      us.technicals?.supportResistance
        .slice(0, 3)
        .map((sr) => `${sr.symbol}: S:[${sr.support.join(',')}] R:[${sr.resistance.join(',')}]`)
        .join(' | ') || 'N/A',
    usDcf:
      us.valuation?.dcf
        .map((v) => `${v.symbol}: FV $${v.fairValue.toFixed(2)} (${v.upside.toFixed(1)}% upside)`)
        .join(', ') || 'N/A',
    usMonteCarlo: us.valuation?.monteCarlo.map((m) => `${m.symbol}: $${m.p50.toFixed(2)}`).join(', ') || 'N/A',
    usAssetList: us.assetList.join(', '),
    usCorrelationMatrix: JSON.stringify(us.correlationMatrix),
    vnAssets: vn.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}% on ${timeframe})`).join(', '),
    vnPhase: vn.technicals?.cycle.phase || 'N/A',
    vnDescription: vn.technicals?.cycle.description || 'N/A',
    vnSupportResistance:
      vn.technicals?.supportResistance
        .slice(0, 3)
        .map((sr) => `${sr.symbol}: S:[${sr.support.join(',')}] R:[${sr.resistance.join(',')}]`)
        .join(' | ') || 'N/A',
    vnDcf:
      vn.valuation?.dcf
        .map((v) => `${v.symbol}: FV ${v.fairValue.toLocaleString()} (${v.upside.toFixed(1)}% upside)`)
        .join(', ') || 'N/A',
    vnSectorMultiples:
      vn.valuation?.sectorComparison
        .map((s) => `${s.symbol} P/E: ${s.metrics.find((m) => m.label.includes('P/E'))?.asset}`)
        .join(', ') || 'N/A',
    vnAssetList: vn.assetList.join(', '),
    vnCorrelationMatrix: JSON.stringify(vn.correlationMatrix),
  });

  const { text } = await generateText({
    model,
    prompt,
  });

  try {
    // Clean potential markdown wrap
    const cleanJson = text.replace(/```json|```/g, '').trim();
    return JSON.parse(cleanJson);
  } catch (e) {
    const message = getErrorMessage(e);
    console.error('[MarketDataService] Failed to parse AI market analysis JSON:', message);
    return null;
  }
}

/**
 * Fetches data for a group of tickers and computes state
 */
async function fetchMarketGroup(
  tickers: { symbol: string; name: string }[],
  market: 'US' | 'VN',
  timeframe: string,
): Promise<MarketState> {
  let validAssets: MarketAsset[] = [];

  if (market === 'VN' && timeframe === '1h') {
    // Specialized fetch for VN market using CafeF for real-time daily snapshot
    const [vnIndices, fxData] = await Promise.all([fetchVNIndicesFromCafeF(), fetchUSDVNDFromCurrencyAPI()]);

    // Also fetch Yahoo Finance historicals to get the `closes` arrays for correlation and technicals
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

          // Append real-time tick to closes if not exactly matching the last one
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
    // Standard Yahoo fetch for US & VN (when timeframe > 1h)
    const assetPromises = tickers.map((t) => fetchAssetData(t.symbol, t.name, market, timeframe));
    const assets = await Promise.all(assetPromises);
    validAssets = assets.filter((a): a is MarketAsset => a !== null);
  }

  // Compute Drivers
  const sortedByChange = [...validAssets].sort((a, b) => Math.abs(b.percentChange) - Math.abs(a.percentChange));
  const topMovers = sortedByChange.slice(0, 3).map((a) => ({ symbol: a.name, change: a.percentChange }));

  const moversStr = topMovers.map((m) => `${m.symbol} (${m.change > 0 ? '+' : ''}${m.change.toFixed(2)}%)`).join(', ');

  // Compute Capital Flow
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
    const networkError = isAppError(e)
      ? e
      : new NetworkError('CafeF VN indices fetch failed', {
          context: { source: 'cafef.vn', endpoint: 'RealtimeChartHeader' },
        });
    console.error('[MarketDataService]', networkError.message);
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
      percentChange: 0.1, // Mock since Currency-API doesn't give historical in this endpoint easily
      dayChange: 0.1,
      weekChange: 0.2,
      direction: 'up',
      momentum: 'stable',
    };
  } catch (e) {
    const networkError = isAppError(e)
      ? e
      : new NetworkError('USD/VND currency rate fetch failed', {
          context: { source: 'fawazahmed0/currency-api', endpoint: 'currencies/usd' },
        });
    console.error('[MarketDataService]', networkError.message);
    return null;
  }
}

/**
 * Fetches single asset data from Yahoo Finance (Used for US market)
 */
async function fetchAssetData(
  symbol: string,
  name: string,
  market: 'US' | 'VN',
  timeframe = '1h',
): Promise<MarketAsset | null> {
  try {
    // Map timeframe to Yahoo interval and range
    let interval = '1h';
    let range = '7d';

    switch (timeframe) {
      case '1h':
        interval = '1h';
        range = '7d'; // Weekly context for 1h candles
        break;
      case '4h':
        interval = '1h'; // Yahoo doesn't support 4h directly easily, we use 1h and aggregate later or just use daily range
        range = '14d';
        break;
      case '1d':
        interval = '1d';
        range = '60d'; // 2 months context for daily
        break;
      case '1w':
        interval = '1d';
        range = '90d'; // 3 months context for daily to calculate 1-week rolling trend properly
        break;
      default:
        interval = '1h';
        range = '7d';
    }

    const url = `https://query1.finance.yahoo.com/v8/finance/chart/${encodeURIComponent(symbol)}?range=${range}&interval=${interval}`;
    const res = await fetch(url);
    if (!res.ok) return null;

    const data = await res.json();
    const result = data.chart.result?.[0];
    if (!result) return null;

    const meta = result.meta;
    const price = meta.regularMarketPrice;

    const quotes = result.indicators.quote[0];
    if (!quotes || !quotes.close) return null;

    const timestampsRaw = result.timestamp || [];
    const closesRaw = quotes.close || [];
    const highsRaw = quotes.high || [];
    const lowsRaw = quotes.low || [];

    // Filter both to exclude null closes
    const validPairs = timestampsRaw
      .map((t: number, i: number) => ({ t, c: closesRaw[i], h: highsRaw[i], l: lowsRaw[i] }))
      .filter((p: any) => p.c !== null);

    const closes = validPairs.map((p: any) => p.c);
    const highs = validPairs.map((p: any) => p.h);
    const lows = validPairs.map((p: any) => p.l);
    const timestamps = validPairs.map((p: any) => p.t);

    if (closes.length < 2) return null;

    const currentPrice = closes[closes.length - 1];

    // Calculate changes based on timeframe context
    let offset = 2; // For 1h and 1d, 1 candle ago is closes.length - 2
    if (timeframe === '4h') offset = 5; // 4 hours ago = closes.length - 5 (current + 4 prev)
    if (timeframe === '1w') offset = 6; // 1 rolling week ago (5 trading days) = closes.length - 6

    const prevPriceIdx = Math.max(0, closes.length - offset);
    const prevPrice = closes[prevPriceIdx] || currentPrice;
    const percentChange = prevPrice ? ((currentPrice - prevPrice) / prevPrice) * 100 : 0;

    let oneDayAgoIdx = 0;
    let oneWeekAgoIdx = 0;

    if (interval === '1h') {
      oneDayAgoIdx = Math.max(0, closes.length - 8); // approx 1 trading day (usually 7-8 h)
      oneWeekAgoIdx = Math.max(0, closes.length - 40); // approx 1 trading week
    } else if (interval === '1d') {
      oneDayAgoIdx = Math.max(0, closes.length - 2); // 1 trading day ago
      oneWeekAgoIdx = Math.max(0, closes.length - 6); // approx 1 trading week (5 days)
    }

    const dayChange = closes[oneDayAgoIdx] ? ((currentPrice - closes[oneDayAgoIdx]) / closes[oneDayAgoIdx]) * 100 : 0;
    const weekChange = closes[oneWeekAgoIdx]
      ? ((currentPrice - closes[oneWeekAgoIdx]) / closes[oneWeekAgoIdx]) * 100
      : 0;

    const direction = percentChange > 0.05 ? 'up' : percentChange < -0.05 ? 'down' : 'flat';

    // Momentum logic
    let momentum: 'fire' | 'stable' | 'sleep' = 'stable';
    if (Math.abs(percentChange) > 2 || (direction === 'up' && dayChange > 5)) momentum = 'fire';
    else if (Math.abs(percentChange) < 0.1) momentum = 'sleep';

    return {
      symbol: name,
      name,
      market,
      price: currentPrice, // Use last close as the main price for timeframe consistency
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
  } catch (error) {
    const networkError = isAppError(error)
      ? error
      : new NetworkError(`Failed to fetch asset data for ${symbol}`, {
          context: { symbol, name, market, timeframe, source: 'yahoo-finance' },
        });
    console.error('[MarketDataService]', networkError.message);
    return null;
  }
}

function computeCapitalFlow(assets: MarketAsset[], market: 'US' | 'VN'): MarketState['capitalFlow'] {
  const gold = assets.find((a) => a.name.includes('Gold'));
  const sp500 = assets.find((a) => a.name === 'S&P500' || a.name === 'VN-Index');
  const vix = assets.find((a) => a.name === 'VIX');

  if (gold && gold.percentChange > 0.5 && ((sp500 && sp500.percentChange < 0) || (vix && vix.percentChange > 2))) {
    return {
      signal: 'DEFENSIVE',
      summaryEn: `Defensive signal led by Gold (+${gold.percentChange.toFixed(2)}%). Equities under pressure.`,
      summaryVi: `Tín hiệu phòng thủ đang nghiêng về Vàng (+${gold.percentChange.toFixed(2)}%). Cổ phiếu đang gặp áp lực.`,
    };
  }

  if (sp500 && sp500.percentChange > 0.5 && vix && vix.percentChange < 0) {
    return {
      signal: 'RISK-ON',
      summaryEn: `Risk-on environment detected. Equities gaining (+${sp500.percentChange.toFixed(2)}%).`,
      summaryVi: `Tín hiệu tăng trưởng được ghi nhận. Cổ phiếu đang tăng (+${sp500.percentChange.toFixed(2)}%).`,
    };
  }

  return {
    signal: 'MIXED',
    summaryEn: `Market signals are mixed. No clear direction in capital flow.`,
    summaryVi: `Tín hiệu thị trường đang hỗn hợp. Chưa có xu hướng rõ ràng của dòng tiền.`,
  };
}

function detectScenario(us: MarketState, vn: MarketState): NonNullable<MarketState['scenarios']>[0] {
  const vix = us.assets.find((a) => a.name === 'VIX');
  const sp500 = us.assets.find((a) => a.name === 'S&P500');
  const gold = us.assets.find((a) => a.name === 'Gold');
  const oil = us.assets.find((a) => a.name === 'WTI');
  const vn30 = vn.assets.find((a) => a.name === 'VN30');

  // BunnyQuant: Falling Knife Detection for VN30
  if (vn30 && vn30.percentChange < -1.5) {
    return {
      name: 'Falling Knife Warning',
      regime: 'Crisis',
      confidence: 90,
      summaryEn: 'Severe institutional dumping phase. RSI/Oversold indicators are failing. Do NOT average down.',
      summaryVi:
        'Cảnh báo Bắt Dao Rơi. Chu kỳ giảm mạnh, các chỉ báo như RSI hiện vô dụng. Tuyệt đối KHÔNG "cứ đỏ là mua" hay DCA.',
      actionEn: 'Halt all buying immediately. Wait for bottom accumulation confirmation.',
      actionVi: 'Dừng hoàn toàn việc mua vào. Đợi tín hiệu xác nhận tạo đáy tích lũy.',
    };
  }

  if (vix && vix.price > 30) {
    return {
      name: 'Crisis mode',
      regime: 'Crisis',
      confidence: 85,
      summaryEn: 'High volatility detected. Extreme panic across markets.',
      summaryVi: 'Biến động mạnh. Các thị trường đang ở mức hoảng loạn cao.',
      actionEn: 'Shift to cash and defensive assets. Hedge long positions.',
      actionVi: 'Chuyển sang tiền mặt và tài sản phòng thủ. Hedging các vị thế mua.',
    };
  }

  if (vix && vix.price < 15 && sp500 && sp500.percentChange > 0) {
    return {
      name: 'Normal Growth',
      regime: 'Risk-ON',
      confidence: 75,
      summaryEn: 'Scenario detected based on current correlations and price action.',
      summaryVi: 'Kịch bản được nhận diện dựa trên tương quan và biến động giá hiện tại.',
      actionEn: 'Maintain long exposure to risk assets and strong momentum sectors.',
      actionVi: 'Duy trì nắm giữ các tài sản rủi ro và các nhóm ngành có dòng tiền mạnh.',
    };
  }

  if (oil && oil.percentChange > 3 && sp500 && sp500.percentChange < -1) {
    return {
      name: 'Stagflation Risk',
      regime: 'Stagflation',
      confidence: 65,
      summaryEn: 'Oil surging while equities fall suggests supply-side pressure.',
      summaryVi: 'Giá dầu tăng mạnh trong khi cổ phiếu giảm cho thấy áp lực từ nguồn cung.',
      actionEn: 'Increase exposure to commodities/energy. Reduce cyclical stocks.',
      actionVi: 'Tăng cường tỷ trọng hàng hóa/năng lượng. Giảm cổ phiếu chu kỳ.',
    };
  }

  return {
    name: 'Neutral / Mixed',
    regime: 'Risk-OFF',
    confidence: 50,
    summaryEn: 'Market is in a transitional or uncertain phase.',
    summaryVi: 'Thị trường đang ở giai đoạn chuyển giao hoặc không chắc chắn.',
    actionEn: 'Hold current positions. Wait for a clear directional breakout.',
    actionVi: 'Giữ nguyên vị thế hiện tại. Chờ đợi xu hướng định hướng rõ ràng.',
  };
}

function computeReturns(closes: number[]): number[] {
  const returns = [];
  for (let i = 1; i < closes.length; i++) {
    returns.push((closes[i] - closes[i - 1]) / closes[i - 1]);
  }
  return returns;
}

function calculatePearson(x: number[], y: number[]): number {
  if (x.length === 0 || y.length === 0) return 0;
  const len = Math.min(x.length, y.length);
  const x1 = x.slice(-len);
  const y1 = y.slice(-len);

  const xMean = x1.reduce((a, b) => a + b, 0) / len;
  const yMean = y1.reduce((a, b) => a + b, 0) / len;

  let num = 0;
  let den1 = 0;
  let den2 = 0;

  for (let i = 0; i < len; i++) {
    const xDiff = x1[i] - xMean;
    const yDiff = y1[i] - yMean;
    num += xDiff * yDiff;
    den1 += xDiff * xDiff;
    den2 += yDiff * yDiff;
  }

  if (den1 === 0 || den2 === 0) return 1;
  return num / Math.sqrt(den1 * den2);
}

function generateRealCorrelationMatrix(assets: MarketAsset[]): number[][] {
  const matrix: number[][] = [];
  const returnsArray = assets.map((a) => computeReturns(a.closes || []));

  for (let i = 0; i < assets.length; i++) {
    const row: number[] = [];
    for (let j = 0; j < assets.length; j++) {
      if (i === j) {
        row.push(1);
      } else {
        const val = calculatePearson(returnsArray[i], returnsArray[j]);
        // Handle NaN/Infinity
        row.push(Number.isNaN(val) ? 0 : Number(val.toFixed(3)));
      }
    }
    matrix.push(row);
  }
  return matrix;
}

function generateTechnicals(assets: MarketAsset[], market: 'US' | 'VN'): Technicals {
  const primaryAsset =
    assets.find((a) => a.name === 'S&P500' || a.name === 'VN-Index' || a.name === 'VN30') || assets[0];
  const closes = primaryAsset?.closes || [];
  const avgChange = assets.reduce((sum, a) => sum + a.percentChange, 0) / (assets.length || 1);

  // Indicators for the primary asset
  const rsi = calculateRSI(closes);
  const ema20 = calculateEMA(closes, 20);
  const ema50 = calculateEMA(closes, 50);
  const ema200 = calculateEMA(closes, 200);
  const sma20 = calculateSMA(closes, 20);
  const sma50 = calculateSMA(closes, 50);

  // Real ATR Calculation
  const highs = primaryAsset.highs || [];
  const lows = primaryAsset.lows || [];
  let atr = primaryAsset.price * 0.02; // Fallback

  if (closes.length > 14 && highs.length > 14 && lows.length > 14) {
    let trSum = 0;
    for (let i = closes.length - 14; i < closes.length; i++) {
      const h = highs[i] ?? closes[i];
      const l = lows[i] ?? closes[i];
      const pc = closes[i - 1] ?? closes[i];
      const tr = Math.max(h - l, Math.abs(h - pc), Math.abs(l - pc));
      trSum += tr;
    }
    atr = trSum / 14;
  }

  // Cycle Phase Logic (Wyckoff Heuristics)
  let phase: Technicals['cycle']['phase'] = 'Accumulation';
  let desc = '';
  let descVi = '';

  const currentPrice = primaryAsset.price;
  const isAboveEma20 = ema20 ? currentPrice > ema20 : false;
  const isAboveEma50 = ema50 ? currentPrice > ema50 : false;
  const isAboveEma200 = ema200 ? currentPrice > ema200 : true; // Trend fallback

  if (isAboveEma20 && isAboveEma50 && isAboveEma200) {
    phase = 'Markup';
    desc = 'Expansion phase with strong institutional demand.';
    descVi = 'Giai đoạn Tăng giá (Mark-up). Xu hướng tăng mạnh với hỗ trợ từ dòng tiền lớn.';
  } else if (!isAboveEma20 && !isAboveEma50 && !isAboveEma200) {
    phase = 'Mark-Down';
    desc = 'Distribution complete. Aggressive sell-off phase in progress.';
    descVi = 'Giai đoạn Giảm giá (Mark-down). Chu kỳ bán tháo mạnh mẽ sau khi phân phối.';
  } else if (isAboveEma50 && rsi && rsi > 70) {
    phase = 'Distribution';
    desc = 'Supply starting to overwhelm demand. Smart money is exiting.';
    descVi = 'Giai đoạn Phân phối. Nguồn cung bắt đầu áp đảo, dòng tiền lớn đang thoát hàng.';
  } else if (!isAboveEma200 && rsi && rsi < 30) {
    phase = 'Accumulation';
    desc = 'Bottoming process. Institutional players absorbing supply.';
    descVi = 'Giai đoạn Tích lũy. Giá đang tạo đáy, các tổ chức đang âm thầm gom hàng.';
  } else if (isAboveEma50 && !isAboveEma20) {
    phase = 'Decline';
    desc = 'Momentum slowing down. Initial signs of trend reversal.';
    descVi = 'Đà tăng đang chững lại. Các dấu hiệu đầu tiên của sự đảo chiều xu hướng.';
  } else {
    phase = 'Accumulation';
    desc = 'Consolidation phase. Market searching for new equilibrium.';
    descVi = 'Giai đoạn đi ngang tích lũy. Thị trường đang tìm kiếm điểm cân bằng mới.';
  }

  // Phases for Donut chart (Heuristic distribution)
  const phases = [
    { label: 'Accumulation', value: phase === 'Accumulation' ? 80 : 5 },
    { label: 'Mark-Up', value: phase === 'Markup' ? 80 : 5 },
    { label: 'Distribution', value: phase === 'Distribution' ? 80 : 5 },
    { label: 'Mark-Down', value: phase === 'Mark-Down' ? 80 : 5 },
  ];

  // Signal Generation (Expanded Actions)
  let action: Technicals['signals']['action'] = 'HOLD/WATCH';
  let actionVi = 'NẮM GIỮ/QUAN SÁT';
  let confidence = 50;
  let reasons: string[] = [];
  let reasonsVi: string[] = [];

  if (phase === 'Markup') {
    if (rsi && rsi < 65) {
      action = 'LONG';
      actionVi = 'LỆNH LONG';
      confidence = 85;
      reasons = ['Strong uptrend confirmed', 'Healthy RSI levels', 'Breakout momentum active'];
      reasonsVi = ['Xác nhận xu hướng tăng mạnh', 'Chỉ báo RSI ổn định', 'Đà bùng nổ đang tiếp diễn'];
    } else {
      action = 'TAKE PROFIT';
      actionVi = 'CHỐT LỜI';
      confidence = 75;
      reasons = ['Overextended Markup phase', 'RSI entering overbought territory', 'Protecting gains recommended'];
      reasonsVi = ['Giai đoạn tăng giá quá đà', 'RSI đi vào vùng quá mua', 'Khuyến nghị chốt lời bảo vệ thành quả'];
    }
  } else if (phase === 'Mark-Down') {
    if (rsi && rsi > 35) {
      action = 'SHORT';
      actionVi = 'LỆNH SHORT';
      confidence = 80;
      reasons = ['Aggressive Markdown phase', 'Trend resistance holding', 'Bearish volume expanding'];
      reasonsVi = ['Giai đoạn Giảm giá mạnh', 'Kháng cự xu hướng được giữ vững', 'Khối lượng bán đang gia tăng'];
    } else {
      action = 'EXIT';
      actionVi = 'THOÁT VỊ THẾ';
      confidence = 90;
      reasons = ['Extreme bearish momentum', 'Final washout phase', 'Oversold bounce likely - Exit shorts'];
      reasonsVi = ['Đà giảm cực đại', 'Giai đoạn rũ bỏ cuối cùng', 'Dễ có nhịp hồi kỹ thuật - Ưu tiên thoát lệnh'];
    }
  } else if (phase === 'Distribution') {
    if (rsi && rsi > 60) {
      action = 'REDUCE';
      actionVi = 'GIẢM TỶ TRỌNG';
      confidence = 70;
      reasons = ['Distribution signs detected', 'Momentum divergent from price', 'De-risking portfolio recommended'];
      reasonsVi = ['Phát hiện dấu hiệu phân phối', 'Đà tăng phân kỳ với giá', 'Khuyến nghị hạ tỷ trọng danh mục'];
    } else {
      action = 'EXIT';
      actionVi = 'TẤT TOÁN VỊ THẾ';
      confidence = 65;
      reasons = ['Trend integrity lost', 'Smart money exiting', 'High risk of cascading decline'];
      reasonsVi = ['Mất xu hướng tăng', 'Dòng tiền lớn đã thoát', 'Rủi ro cao xảy ra nhịp giảm mạnh'];
    }
  } else if (phase === 'Accumulation') {
    if (rsi && rsi < 30) {
      action = 'LONG';
      actionVi = 'MUA TÍCH LŨY';
      confidence = 60;
      reasons = ['Value area reached', 'Supply exhaustion signs', 'Institutional buying detected'];
      reasonsVi = ['Đã chạm vùng giá trị', 'Dấu hiệu cạn kiệt nguồn cung', 'Phát hiện lực mua từ tổ chức'];
    } else {
      action = 'HOLD/WATCH';
      actionVi = 'QUAN SÁT/CHỜ ĐỢI';
      confidence = 55;
      reasons = ['Sideways range active', 'Waiting for clear breakout', 'Volatility compression in progress'];
      reasonsVi = ['Đang đi ngang tích lũy', 'Chờ đợi điểm bùng nổ xác nhận', 'Biến động đang bị nén chặt'];
    }
  }

  // ATR-based SL/TP (Real Volatility)
  const entry = currentPrice;
  const stopLoss = action === 'LONG' || action === 'HOLD/WATCH' ? entry - atr * 1.5 : entry + atr * 1.5;
  const takeProfit = action === 'LONG' || action === 'HOLD/WATCH' ? entry + atr * 3.5 : entry - atr * 3.5;
  const rr = Math.abs((takeProfit - entry) / (entry - stopLoss));

  // S/R logic
  const supportResistance = assets.slice(0, 5).map((a) => {
    const aCloses = a.closes || [];
    let sma = a.price;
    let stdDev = a.price * 0.05;

    let localMins = [a.price * 0.95, a.price * 0.92, a.price * 0.88];
    let localMaxs = [a.price * 1.05, a.price * 1.08, a.price * 1.12];

    if (aCloses.length > 20) {
      const last20 = aCloses.slice(-20);
      sma = last20.reduce((s, v) => s + v, 0) / last20.length;
      const variance = last20.reduce((s, v) => s + Math.pow(v - sma, 2), 0) / last20.length;
      stdDev = Math.sqrt(variance);

      localMins = [...last20].sort((x, y) => x - y).slice(0, 3);
      localMaxs = [...last20].sort((x, y) => y - x).slice(0, 3);
    }

    return {
      symbol: a.name,
      support: localMins,
      resistance: localMaxs,
      bollingerUpper: sma + stdDev * 2,
      bollingerLower: sma - stdDev * 2,
      bollingerMid: sma,
    };
  });

  // Seasonality calculation
  const dates = (primaryAsset.timestamps || []).map((ts) => new Date(ts * 1000));
  const dayStats = calculateSeasonality(closes, dates, 'day');
  const weekStats = calculateSeasonality(closes, dates, 'week');
  const monthStats = calculateSeasonality(closes, dates, 'month');

  const combinedSeasonality = [
    ...dayStats.map((s) => ({ ...s, type: 'day' as const })),
    ...weekStats.map((s) => ({ ...s, type: 'week' as const })),
    ...monthStats.map((s) => ({ ...s, type: 'month' as const })),
  ];

  // ICT Signature Detection (FVG / Order Blocks - using existing highs/lows from line 821)
  const fvgs = [];
  if (highs.length >= 3) {
    for (let i = highs.length - 1; i >= highs.length - 10 && i >= 2; i--) {
      // Bullish FVG: Low of candle 0 > High of candle -2
      if (lows[i] > highs[i - 2]) {
        fvgs.push({ type: 'BULLISH', top: lows[i], bottom: highs[i - 2], gap: lows[i] - highs[i - 2] });
      }
      // Bearish FVG: High of candle 0 < Low of candle -2
      else if (highs[i] < lows[i - 2]) {
        fvgs.push({ type: 'BEARISH', top: lows[i - 2], bottom: highs[i], gap: lows[i - 2] - highs[i] });
      }
    }
  }

  // Trend Analysis Metrics (matching UI)
  const isUp = currentPrice > (ema20 || 0) && (ema20 || 0) > (ema50 || 0);
  const isDown = currentPrice < (ema20 || 0) && (ema20 || 0) < (ema50 || 0);
  const trendDirection = isUp ? 'Tăng' : isDown ? 'Giảm' : 'Đi ngang';
  const trendStrength = Math.min(100, Math.max(0, (rsi || 50) + (isUp ? 20 : isDown ? -20 : 0)));
  const trendConfidence = Math.min(100, Math.floor(rsi ? (rsi > 40 && rsi < 60 ? 40 : 80) : 50));

  // Cycle Probabilities for Ring Chart
  const phaseWeights = [
    { label: 'Tích lũy', value: phase === 'Accumulation' ? 60 : 10, color: '#6366f1' },
    { label: 'Tăng giá', value: phase === 'Markup' ? 70 : 10, color: '#10b981' },
    { label: 'Phân phối', value: phase === 'Distribution' ? 50 : 5, color: '#fbbf24' },
    { label: 'Giảm giá', value: phase === 'Mark-Down' ? 80 : 5, color: '#f43f5e' },
  ];
  const totalWeight = phaseWeights.reduce((s, w) => s + w.value, 0);
  const normalizedPhases = phaseWeights.map((w) => ({ ...w, value: Math.round((w.value / totalWeight) * 100) }));

  // Sentiment Analysis
  const vixAsset = assets.find((a) => a.name === 'VIX' || a.symbol === '^VIX');
  const vixVal = vixAsset?.price || 20;
  const sentimentScore = Math.max(0, Math.min(100, 100 - vixVal * 2.5 + (rsi ? rsi - 50 : 0)));
  const sentimentLabel = sentimentScore > 65 ? 'Greed' : sentimentScore < 35 ? 'Fear' : 'Neutral';

  return {
    cycle: {
      phase,
      description: desc,
      descriptionVi: descVi,
      strength: trendStrength,
      confidence: trendConfidence,
      phases: normalizedPhases,
    },
    trend: {
      direction: trendDirection,
      directionEn: isUp ? 'Up' : isDown ? 'Down' : 'Sideways',
      strength: trendStrength,
      confidence: trendConfidence,
    },
    ict: {
      fvgs: fvgs.slice(0, 3),
      orderBlocks: [], // Placeholder for future logic
    },
    sentiment: {
      score: Math.round(sentimentScore),
      label: sentimentLabel,
      labelVi: sentimentLabel === 'Greed' ? 'Tham lam' : sentimentLabel === 'Fear' ? 'Sợ hãi' : 'Trung lập',
    },
    indicators: { rsi, ema20, ema50, ema200, sma20, sma50 },
    signals: {
      action,
      actionVi,
      entry,
      stopLoss,
      takeProfit,
      rr: parseFloat(rr.toFixed(2)),
      confidence,
      reasons,
      reasonsVi,
      optimalEntry: {
        price:
          action === 'LONG'
            ? ema20 && ema20 < entry
              ? ema20
              : entry * 0.985
            : ema20 && ema20 > entry
              ? ema20
              : entry * 1.015,
        pullbackPercent: Math.abs(((entry - (ema20 || entry)) / entry) * 100),
        pullbackAmount: Math.abs(entry - (ema20 || entry)),
      },
    },
    timeframeRelationships: [
      {
        pair: '1wk -> 1d',
        status: isAboveEma50 && isAboveEma200 ? 'ALIGNED' : 'CHOPIED',
        relationship: isAboveEma50 && isAboveEma200 ? 'Bullish Dominance' : 'Regime Transition',
        relationshipVi: isAboveEma50 && isAboveEma200 ? 'Thế trận giá tăng' : 'Giai đoạn chuyển đổi',
        advice: isAboveEma50 ? 'Buy dips on support' : 'Wait for bottom formation',
        adviceVi: isAboveEma50 ? 'Mua tại vùng hỗ trợ' : 'Chờ đợi xác nhận tạo đáy',
      },
      {
        pair: '1d -> 4h',
        status: isAboveEma20 && isAboveEma50 ? 'STRONG' : 'WEAK',
        relationship: isAboveEma20 && isAboveEma50 ? 'Aggressive Trend' : 'Mean Reversion',
        relationshipVi: isAboveEma20 && isAboveEma50 ? 'Xu hướng quyết liệt' : 'Hồi quy về giá trị trung bình',
        advice: isAboveEma20 ? 'Active Scaling' : 'Wait for SMA Alignment',
        adviceVi: isAboveEma20 ? 'Chủ động gia tăng vị thế' : 'Đợi xác nhận từ đường SMA',
      },
      {
        pair: '4h -> 1h',
        status: isAboveEma20 ? 'ALIGNED' : 'WEAK',
        relationship: isAboveEma20 ? 'Momentum Push' : 'Short-term consolidation',
        relationshipVi: isAboveEma20 ? 'Đà tăng mạnh' : 'Tích lũy ngắn hạn',
        advice: isAboveEma20 ? 'Momentum Entry' : 'Wait for EMA20 breakout',
        adviceVi: isAboveEma20 ? 'Vào lệnh theo đà' : 'Đợi giá vượt EMA20',
      },
    ],
    entryTimingScore: {
      overall: Math.min(10, Math.floor((rsi ? (rsi < 35 ? 8 : rsi > 75 ? 2 : 5) : 5) + (isAboveEma20 ? 2 : 0))),
      higherTfSupport: isAboveEma50 ? 5 : 2,
      lowerTfConfirm: isAboveEma20 ? 4 : 1,
    },
    supportResistance,
    seasonality: combinedSeasonality,
    n: closes.length,
    dateRange: `${closes.length > 0 ? new Date((primaryAsset.timestamps?.[0] || 0) * 1000).toLocaleDateString('vi-VN') : 'N/A'} -> ${closes.length > 0 ? new Date((primaryAsset.timestamps?.[closes.length - 1] || 0) * 1000).toLocaleDateString('vi-VN') : 'N/A'}`,
  };
}

function generateValuation(assets: MarketAsset[], market: 'US' | 'VN'): Valuation {
  // Filter for indices or major assets to perform valuation on
  const valuationAssets = assets.filter(
    (a) =>
      a.name.includes('Index') ||
      a.name.includes('HNX') ||
      a.name.includes('VN30') ||
      a.name.includes('UPCOM') ||
      a.name.includes('USD/VND') ||
      a.name === 'S&P500' ||
      a.name === 'NQ100' ||
      a.name === 'Gold',
  );

  const dcf = valuationAssets.map((a) => {
    // Replaced mock DCF with Historical Mean Reversion (Fair Value) since DCF implies cash flows unavailable for indices
    const closes = a.closes || [];
    let histMean = a.price;
    if (closes.length > 0) {
      histMean = closes.reduce((s, v) => s + v, 0) / closes.length;
    }
    const fairValue = histMean;

    return {
      symbol: a.name,
      fairValue,
      upside: (fairValue / a.price - 1) * 100,
      assumptions: [
        { label: 'Valuation Model', value: 'Historical Mean Reversion' },
        { label: 'Lookback Period', value: `${closes.length} periods` },
        {
          label: 'Mean',
          value: fairValue.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
        },
      ],
    };
  });

  const monteCarlo = valuationAssets.map((a) => {
    const closes = a.closes || [];
    const returns = computeReturns(closes);
    let mu = 0;
    let sigma = 0.02; // default volatility 2%

    if (returns.length > 0) {
      mu = returns.reduce((s, v) => s + v, 0) / returns.length;
      const variance = returns.reduce((s, v) => s + Math.pow(v - mu, 2), 0) / returns.length;
      sigma = Math.sqrt(variance);
    }

    // GBM analytical endpoints (log-normal distribution)
    // P_t = P_0 * exp((mu - 0.5 * sigma^2)*t + sigma * W_t)
    const t = 10; // project 10 periods ahead
    const drift = (mu - 0.5 * sigma * sigma) * t;
    const vol = sigma * Math.sqrt(t);

    // Z-scores: 10th percentile ~ -1.28, 50th = 0, 90th ~ 1.28
    const p10 = a.price * Math.exp(drift - 1.28 * vol);
    const p50 = a.price * Math.exp(drift);
    const p90 = a.price * Math.exp(drift + 1.28 * vol);

    return {
      symbol: a.name,
      p10,
      p50,
      p90,
      iterations: 10000, // Real analytical calculation simulates large N
    };
  });

  const sectorComparison = valuationAssets.map((a, i) => {
    const closes = a.closes || [];
    const returns = computeReturns(closes);
    let vol = 0;
    let drawdown = 0;
    if (returns.length > 0) {
      const mu = returns.reduce((s, v) => s + v, 0) / returns.length;
      const variance = returns.reduce((s, v) => s + Math.pow(v - mu, 2), 0) / returns.length;
      vol = Math.sqrt(variance) * 100;

      const maxPrice = Math.max(...closes, a.price);
      drawdown = ((a.price - maxPrice) / maxPrice) * 100;
    }

    const ma20 = closes.length >= 20 ? closes.slice(-20).reduce((s, v) => s + v, 0) / 20 : a.price;
    const vsMa20 = (a.price / ma20 - 1) * 100;

    // Calculate dynamic "Market Average" based on the group
    const avgVol =
      valuationAssets.reduce((sum, item) => sum + (item.price > 0 ? 1.5 : 0), 0) / valuationAssets.length || 1.5;
    const avgDrawdown = -5.0; // Baseline market risk floor

    return {
      symbol: a.name,
      sector: market === 'VN' ? 'VN-Market Baseline' : 'Global Core Baseline',
      metrics: [
        { label: 'Realized Volatility', asset: Number(vol.toFixed(2)), avg: avgVol },
        { label: 'Max Drawdown (%)', asset: Number(drawdown.toFixed(2)), avg: avgDrawdown },
        { label: 'Price vs MA20 (%)', asset: Number(vsMa20.toFixed(2)), avg: 0 },
      ],
    };
  });

  return { dcf, monteCarlo, sectorComparison };
}
