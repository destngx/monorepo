import { NextResponse } from 'next/server';
import { invalidateCache } from '@wealth-management/utils';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST() {
  try {
    await invalidateCache('accounts');
    await invalidateCache('transactions');
    await invalidateCache('budget');

    return NextResponse.json({ success: true, message: 'Cache cleared successfully' });
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('Sync API Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
