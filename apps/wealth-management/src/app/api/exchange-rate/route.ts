import { NextResponse } from 'next/server';
import { getExchangeRate } from '@wealth-management/services/server';
import { NetworkError, isAppError } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const rate = await getExchangeRate();
    return NextResponse.json({ rate });
  } catch (error) {
    if (isAppError(error)) {
      // Still return fallback rate for resilience, but log the structured error
      console.error('[Exchange Rate API Error]', error.toResponse());
    } else {
      const appError = new NetworkError(error instanceof Error ? error.message : 'Failed to fetch exchange rate');
      console.error('[Exchange Rate API Error]', appError.toResponse());
    }
    return NextResponse.json({ rate: 25400 });
  }
}
