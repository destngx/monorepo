import { NextResponse } from 'next/server';
import { getTransactions } from '@wealth-management/services/server';
import { AppError, isAppError } from '@wealth-management/utils/errors';

export async function GET() {
  try {
    const transactions = await getTransactions();
    // Collect all unique tags across all transactions, sorted alphabetically
    const tagSet = new Set<string>();
    for (const txn of transactions) {
      for (const tag of txn.tags ?? []) {
        if (tag) tagSet.add(tag.trim().toLowerCase());
      }
    }
    return NextResponse.json(Array.from(tagSet).sort());
  } catch (error) {
    if (isAppError(error)) {
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError({
      message: error instanceof Error ? error.message : 'Failed to fetch tags',
    });
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
