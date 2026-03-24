import { NextRequest, NextResponse } from 'next/server';
import {
  fetchAssetData,
  generateTechnicals,
  generateValuation,
} from '@wealth-management/services/services/market-data-service';

export async function POST(req: NextRequest) {
  try {
    const { symbol, name, market } = await req.json();

    if (!symbol || !market) {
      return NextResponse.json({ error: 'Symbol and market are required' }, { status: 400 });
    }

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

    return NextResponse.json({
      technicals1h,
      technicals1d,
      valuation,
    });
  } catch (error: any) {
    console.error('Ticker analysis failed:', error);
    const statusCode = error.statusCode || 500;
    return NextResponse.json({ error: error.message }, { status: statusCode });
  }
}
