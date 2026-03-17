import { NextResponse } from 'next/server';
import { getMarketPulseData } from '@wealth-management/services/server';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const timeframe = (searchParams.get('timeframe') as '1h' | '4h' | '1d' | '1w') || '1h';
    const shouldForce = searchParams.get('force') === 'true';

    const data = await getMarketPulseData(timeframe, shouldForce);
    return NextResponse.json(data);
  } catch (error) {
    console.error('[MarketPulseAPI] Error:', error);
    return NextResponse.json({ error: 'Failed to fetch market pulse data' }, { status: 500 });
  }
}
