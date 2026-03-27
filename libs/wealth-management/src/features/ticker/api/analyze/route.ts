import { NextRequest, NextResponse } from 'next/server';
import { getCached, setCache } from '@wealth-management/utils';
import {
  fetchAssetData,
  generateTechnicals,
  generateValuation,
} from '@wealth-management/services/services/market-data-service';

const ANALYZE_CACHE_TTL = 14 * 24 * 3600;

export async function POST(req: NextRequest) {
  try {
    const { symbol, name, market } = await req.json();

    if (!symbol || !market) {
      return NextResponse.json({ error: 'Symbol and market are required' }, { status: 400 });
    }

    const cacheKey = `ticker-analyze:${symbol}:${market}`;

    const cached = await getCached(cacheKey);
    if (cached) {
      console.log(`[TickerAnalyzeAPI] ✓ Cache hit for analysis: ${symbol} (${market})`);
      return NextResponse.json(cached, {
        headers: {
          'Cache-Control': 'public, max-age=2592000, stale-while-revalidate=604800',
          'X-Cache': 'HIT',
        },
      });
    }

    console.log(`[TickerAnalyzeAPI] Cache miss for ${symbol}, generating analysis...`);

    const [asset1h, asset1d] = await Promise.all([
      fetchAssetData(symbol, name || symbol, market, '1h'),
      fetchAssetData(symbol, name || symbol, market, '1d'),
    ]);

    if (!asset1h || !asset1d) {
      return NextResponse.json({ error: 'Failed to fetch asset data for ' + symbol }, { status: 404 });
    }

    const technicals1h = generateTechnicals([asset1h], market);
    const technicals1d = generateTechnicals([asset1d], market);

    const valuation = generateValuation([asset1d], market);

    const analysisResult = {
      technicals1h,
      technicals1d,
      valuation,
    };

    await setCache(cacheKey, analysisResult, ANALYZE_CACHE_TTL);
    console.log(`[TickerAnalyzeAPI] Cached analysis for 14 days: ${symbol} (${market})`);

    return NextResponse.json(analysisResult, {
      headers: {
        'Cache-Control': 'public, max-age=2592000, stale-while-revalidate=604800',
        'X-Cache': 'MISS',
      },
    });
  } catch (error: any) {
    console.error('Ticker analysis failed:', error);
    const statusCode = error.statusCode || 500;
    return NextResponse.json({ error: error.message }, { status: statusCode });
  }
}
