import { NextResponse } from 'next/server';
import { getCategories } from '@wealth-management/services/server';
import { AppError, isAppError } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const categories = await getCategories();
    return NextResponse.json(categories);
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError({
      message: error instanceof Error ? error.message : 'Failed to fetch categories',
    });
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
