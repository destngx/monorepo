import { getCached, setCache } from '@wealth-management/utils';
import { generateText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { loadPrompt, replacePlaceholders } from '../../ai/prompts/loader';
import { NetworkError, isAppError, getErrorMessage } from '../../utils/errors';
import { calculateEMA, calculateSMA, calculateRSI } from '../../utils/technical-analysis';
import { VNStockAdapter, YahooFinanceAdapter, StockDataPoint } from '../data-sources';

const CACHE_PREFIX = 'stock-analysis:';
const ANALYSIS_CACHE_TTL = 30 * 24 * 3600; // 30 days in seconds (2,592,000s)

// Add ±10% jitter to prevent cache stampede
function getAnalysisCacheTTLWithJitter(): number {
  const jitterRange = Math.floor(ANALYSIS_CACHE_TTL * 0.1); // ±10%
  const jitter = Math.floor(Math.random() * (jitterRange * 2)) - jitterRange;
  return ANALYSIS_CACHE_TTL + jitter;
}

export interface StockAnalysis {
  symbol: string;
  name: string;
  price: number;
  change: number;
  market: 'VN';
  technicals: {
    rsi: number;
    ema20: number;
    ema50: number;
    trend: 'BULLISH' | 'BEARISH' | 'NEUTRAL';
    phase: string;
  };
  summary: {
    en: string;
    vi: string;
  };
  recommendation: {
    action: 'BUY' | 'SELL' | 'HOLD' | 'WATCH';
    confidence: number;
    reasonEn: string;
    reasonVi: string;
  };
  lastUpdated: string;
}

/**
 * Service to perform deep stock analysis combining technical indicators and AI synthesis.
 */
export async function getStockAnalysis(symbol: string, forceRefresh = false): Promise<StockAnalysis | null> {
  const cacheKey = `${CACHE_PREFIX}${symbol}`;

  if (!forceRefresh) {
    const cached = await getCached<StockAnalysis>(cacheKey);
    if (cached) return cached;
  }

  try {
    const vnstock = new VNStockAdapter();
    const yahoo = new YahooFinanceAdapter();

    // 1. Fetch metadata and historical data
    const [metadata, history] = await Promise.all([
      vnstock.fetchStock(symbol, 'VN'),
      vnstock
        .fetchHistorical(symbol, 'VN', '1d', '60d')
        .then((res) => res || yahoo.fetchHistorical(symbol, 'VN', '1d', '60d')),
    ]);

    if (!metadata || !history || history.length < 50) {
      console.warn(`[StockAnalysis] Insufficient data for ${symbol}`);
      return null;
    }

    const closes = history.map((p) => p.close);
    const currentPrice = metadata.lastPrice;

    // 2. Compute Technical Indicators
    const rsi = calculateRSI(closes) || 50;
    const ema20 = calculateEMA(closes, 20) || currentPrice;
    const ema50 = calculateEMA(closes, 50) || currentPrice;

    let trend: 'BULLISH' | 'BEARISH' | 'NEUTRAL' = 'NEUTRAL';
    if (currentPrice > ema20 && ema20 > ema50) trend = 'BULLISH';
    else if (currentPrice < ema20 && ema20 < ema50) trend = 'BEARISH';

    // 3. AI Synthesis (Optional: Fallback to heuristics if AI fails)
    let aiResult = null;
    try {
      aiResult = await generateAiStockSummary(symbol, metadata.name, closes, trend, rsi);
    } catch (e) {
      console.error('[StockAnalysis] AI Synthesis failed:', getErrorMessage(e));
    }

    const analysis: StockAnalysis = {
      symbol,
      name: metadata.name,
      price: currentPrice,
      change:
        history.length > 1
          ? ((currentPrice - history[history.length - 2].close) / history[history.length - 2].close) * 100
          : 0,
      market: 'VN',
      technicals: {
        rsi,
        ema20,
        ema50,
        trend,
        phase: aiResult?.phase || (trend === 'BULLISH' ? 'Markup' : trend === 'BEARISH' ? 'Markdown' : 'Accumulation'),
      },
      summary: {
        en: aiResult?.summaryEn || `Stock shows a ${trend.toLowerCase()} trend with RSI at ${rsi.toFixed(1)}.`,
        vi:
          aiResult?.summaryVi ||
          `Cổ phiếu đang trong xu hướng ${trend === 'BULLISH' ? 'tăng' : trend === 'BEARISH' ? 'giảm' : 'đi ngang'} với RSI ở mức ${rsi.toFixed(1)}.`,
      },
      recommendation: {
        action: aiResult?.recommendation?.action || (rsi < 30 ? 'BUY' : rsi > 70 ? 'SELL' : 'HOLD'),
        confidence: aiResult?.recommendation?.confidence || 70,
        reasonEn: aiResult?.recommendation?.reasonEn || 'Based on technical trend and RSI levels.',
        reasonVi: aiResult?.recommendation?.reasonVi || 'Dựa trên xu hướng kỹ thuật và các mức RSI.',
      },
      lastUpdated: new Date().toISOString(),
    };

    await setCache(cacheKey, analysis, getAnalysisCacheTTLWithJitter());
    return analysis;
  } catch (error) {
    const networkError = isAppError(error)
      ? error
      : new NetworkError(`Failed to perform stock analysis for ${symbol}`, {
          context: { symbol, market: 'VN' },
        });
    console.error('[StockAnalysis]', networkError.message);
    return null;
  }
}

/**
 * Uses AI to synthesize technical data into a cohesive stock summary.
 */
async function generateAiStockSummary(symbol: string, name: string, closes: number[], trend: string, rsi: number) {
  const model = getLanguageModel('github-gpt-4o');
  const lastCloses = closes.slice(-10).join(', ');

  const template = await loadPrompt('market', 'stock-analysis');
  if (!template) return null; // Fallback to heuristics

  const prompt = replacePlaceholders(template, {
    symbol,
    name,
    trend,
    rsi: rsi.toFixed(1),
    lastCloses,
  });

  const { text } = await generateText({
    model,
    prompt,
  });

  try {
    const cleanJson = text.replace(/```json|```/g, '').trim();
    return JSON.parse(cleanJson);
  } catch (e) {
    console.error('[StockAnalysis] Failed to parse AI summary JSON');
    return null;
  }
}
