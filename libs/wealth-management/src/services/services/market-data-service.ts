import { getCached, setCache } from '@wealth-management/utils';
import { MarketPulseResponse } from './market-pulse/types';
import { CACHE_PREFIX, PRICE_CACHE_TTL, US_TICKERS, VN_TICKERS } from './market-pulse/constants';
import { fetchMarketGroup } from './market-pulse/data-fetcher';
import { generateRealCorrelationMatrix, generateTechnicals } from './market-pulse/technical-analysis';
import { generateValuation } from './market-pulse/valuation';
import { detectScenario, generateAiMarketAnalysis } from './market-pulse/ai-analysis';

// Re-export functions for consumers (API routes, etc)
export * from './market-pulse/types';
export { fetchAssetData } from './market-pulse/data-fetcher';
export { generateTechnicals } from './market-pulse/technical-analysis';
export { generateValuation } from './market-pulse/valuation';

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
    console.error('[MarketPulse:Gold] Failed to fetch real VN Gold price');
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
    console.error('[MarketPulse:AI] analysis failed');
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
