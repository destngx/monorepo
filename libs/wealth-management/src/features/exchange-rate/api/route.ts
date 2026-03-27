import { NextResponse } from 'next/server';
import { getExchangeRate } from '@wealth-management/services/server';
import { NetworkError, isAppError } from '@wealth-management/utils/errors';
import { getOrSetCache, CACHE_KEYS, CACHE_TTL } from '@wealth-management/utils';

export async function GET() {
  try {
    const rate = await getOrSetCache(
      CACHE_KEYS.EXCHANGE_RATE,
      () => getExchangeRate(),
      CACHE_TTL.EXCHANGE_RATES,
      false,
    );
    return NextResponse.json({ rate });
  } catch (error) {
    if (isAppError(error)) {
      console.error('[Exchange Rate API Error]', error.toResponse());
    } else {
      const appError = new NetworkError(error instanceof Error ? error.message : 'Failed to fetch exchange rate');
      console.error('[Exchange Rate API Error]', appError.toResponse());
    }
    return NextResponse.json({ rate: 25400 });
  }
}
