import { NextResponse } from 'next/server';
import { prefetchAllContent } from '@wealth-management/ai/server';

/**
 * GET /api/init
 * Prefetches all prompts and knowledge from Google Sheets into the in-memory cache.
 * Call this on app startup to warm the cache before AI features are used.
 */
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
    console.error('[Init API Error]:', error);
    const message = error instanceof Error ? error.message : 'Failed to prefetch content from Google Sheets';
    return NextResponse.json({ ready: false, error: message }, { status: 500 });
  }
}
