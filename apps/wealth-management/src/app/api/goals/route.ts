import { NextResponse } from 'next/server';
import { getGoals } from '@wealth-management/services/server';
import { GoalError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const shouldForceFresh = searchParams.get('force') === 'true';

    const goals = await getGoals(shouldForceFresh);
    return NextResponse.json(goals);
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json(error.toResponse(), { status: error.statusCode });
    }
    const message = getErrorMessage(error);
    const appError = new GoalError('Failed to fetch goals', {
      userMessage: 'Unable to retrieve your goals. Please try again.',
      context: { originalError: message },
    });
    return NextResponse.json(appError.toResponse(), { status: appError.statusCode });
  }
}
