import { NextResponse } from 'next/server';
import { getTransactions, addTransaction } from '@wealth-management/services/server';
import { TransactionSchema } from '@wealth-management/schemas';
import { getCategories } from '@wealth-management/services/server';
import { parseDate } from '@wealth-management/utils';
import { handleApiError } from '@wealth-management/utils/server';
import { getOrSetCache, invalidateCachePattern, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const forceFresh = searchParams.get('force') === 'true';

    const [transactions, categories] = await Promise.all([
      getOrSetCache(CACHE_KEYS.TRANSACTIONS, () => getTransactions(false), CACHE_TTL.PORTFOLIO_DATA, forceFresh),
      getOrSetCache('categories:all', () => getCategories(), CACHE_TTL.PORTFOLIO_DATA, forceFresh),
    ]);

    const enrichedTransactions = transactions.map((t) => ({
      ...t,
      categoryType: categories.find((c) => c.name.toLowerCase().trim() === t.category.toLowerCase().trim())?.type,
    }));

    return NextResponse.json(enrichedTransactions);
  } catch (error) {
    return handleApiError(error, 'Transactions');
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();

    if (body.date) {
      body.date = parseDate(String(body.date));
    }

    const validatedData = TransactionSchema.parse(body);

    await addTransaction({
      ...validatedData,
      payment: validatedData.payment ?? null,
      deposit: validatedData.deposit ?? null,
      memo: validatedData.memo ?? null,
      referenceNumber: validatedData.referenceNumber ?? null,
    });

    await invalidateCachePattern('transactions:*');
    await invalidateCachePattern('budget:*');

    return NextResponse.json({ success: true });
  } catch (error) {
    return handleApiError(error, 'Transactions');
  }
}
