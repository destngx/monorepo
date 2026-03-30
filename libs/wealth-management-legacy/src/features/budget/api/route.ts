import { NextResponse } from 'next/server';
import { getBudget } from '@wealth-management/services/server';
import { getCategories } from '@wealth-management/services/server';
import { handleApiError } from '@wealth-management/utils/server';

export async function GET() {
  try {
    const [budget, categories] = await Promise.all([getBudget(), getCategories()]);

    const enrichedBudget = budget.map((b) => ({
      ...b,
      categoryType: categories.find((c) => c.name.toLowerCase().trim() === b.category.toLowerCase().trim())?.type,
    }));

    return NextResponse.json(enrichedBudget);
  } catch (error) {
    return handleApiError(error, 'Budget');
  }
}
