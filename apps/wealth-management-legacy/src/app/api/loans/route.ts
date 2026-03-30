import { NextResponse } from 'next/server';
import { getLoans } from '@wealth-management/services/server';
import { LoanError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const loans = await getLoans();
    return NextResponse.json(loans);
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json(error.toResponse(), { status: error.statusCode });
    }
    const message = getErrorMessage(error);
    const appError = new LoanError('Failed to fetch loans', {
      userMessage: 'Unable to retrieve your loans. Please try again.',
      context: { originalError: message },
    });
    return NextResponse.json(appError.toResponse(), { status: appError.statusCode });
  }
}
