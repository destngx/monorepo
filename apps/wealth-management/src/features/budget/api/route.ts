import { NextResponse } from 'next/server';
import { getBudget } from '@wealth-management/services/server';
import { getCategories } from '@wealth-management/services/server';
import { handleApiError } from '@wealth-management/utils/server';
import { getOrSetCache, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';

export async function GET() {
  try {
    const [budget, categories] = await Promise.all([
      getOrSetCache(CACHE_KEYS.BUDGET, () => getBudget(), CACHE_TTL.PORTFOLIO_DATA, false),
      getOrSetCache('categories:all', () => getCategories(), CACHE_TTL.PORTFOLIO_DATA, false),
    ]);

    const enrichedBudget = budget.map((b) => ({
      ...b,
      categoryType: categories.find((c) => c.name.toLowerCase().trim() === b.category.toLowerCase().trim())?.type,
    }));

    return NextResponse.json(enrichedBudget);
  } catch (error) {
    return handleApiError(error, 'Budget');
  }
}
