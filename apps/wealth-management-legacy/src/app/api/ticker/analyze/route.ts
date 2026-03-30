import { NextRequest, NextResponse } from 'next/server';
import { getCached, setCache } from '@wealth-management/utils';
import {
  fetchAssetData,
  generateTechnicals,
  generateValuation,
} from '@wealth-management/services/services/market-data-service';
import { VNStockAdapter } from '@wealth-management/services/data-sources';

const ANALYZE_CACHE_TTL = 14 * 24 * 3600; // 14 days

export async function POST(req: NextRequest) {
  try {
    const { symbol, name, market } = await req.json();

    if (!symbol || !market) {
      return NextResponse.json({ error: 'Symbol and market are required' }, { status: 400 });
    }

    const cacheKey = `ticker-analyze:${symbol}:${market}`;

    // Check application-code cache first
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

    // Check if vnstock-server is available
    const vnstockAdapter = new VNStockAdapter();
    const isVnstockAvailable = market === 'VN' ? await vnstockAdapter.isServerAvailable() : true;

    const [asset1h, asset1d] = await Promise.all([
      fetchAssetData(symbol, name || symbol, market, '1h'),
      fetchAssetData(symbol, name || symbol, market, '1d'),
    ]);

    if (!asset1h || !asset1d) {
      return NextResponse.json({ error: 'Failed to fetch asset data for ' + symbol }, { status: 404 });
    }

    // Generate technicals
    const technicals1h = generateTechnicals([asset1h], market);
    const technicals1d = generateTechnicals([asset1d], market);

    // Generate valuation using 1D data
    const valuation = generateValuation([asset1d], market);

    // Build data source metadata
    const dataSource = isVnstockAvailable
      ? { provider: 'vnstock-server', status: 'primary' as const }
      : {
          provider: 'yahoo-finance',
          status: 'fallback' as const,
          fallbackReason: 'vnstock-server is unavailable. Using Yahoo Finance as alternative provider.',
        };

    const analysisResult = {
      technicals1h,
      technicals1d,
      valuation,
      dataSource,
    };

    // Cache the analysis for 14 days
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
