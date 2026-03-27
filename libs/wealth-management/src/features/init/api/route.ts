import { NextResponse } from 'next/server';
import { prefetchAllContent } from '@wealth-management/ai/server';
import { AppError, NetworkError } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const result = await prefetchAllContent();

    return NextResponse.json({
      ready: true,
      loaded: {
        prompts: result.prompts,
        knowledge: result.knowledge,
      },
    });
  } catch (error: unknown) {
    const appError =
      error instanceof AppError
        ? error
        : new NetworkError('Failed to prefetch content from Google Sheets', {
            originalError: error instanceof Error ? error.message : String(error),
          });
    console.error('[Init API Error]', appError.toResponse());
    return NextResponse.json({ ready: false, error: appError.userMessage }, { status: appError.statusCode });
  }
}
