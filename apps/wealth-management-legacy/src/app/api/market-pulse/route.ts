import { NextResponse } from 'next/server';
import { getMarketPulseData } from '@wealth-management/services/server';
import { AppError, NetworkError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const timeframe = (searchParams.get('timeframe') as '1h' | '4h' | '1d' | '1w') || '1h';
    const shouldForce = searchParams.get('force') === 'true';

    const data = await getMarketPulseData(timeframe, shouldForce);
    return NextResponse.json(data);
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new NetworkError(getErrorMessage(error), {
      context: { source: 'market-pulse' },
    });
    console.error('[MarketPulseAPI] Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
