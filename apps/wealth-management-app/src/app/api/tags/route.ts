import { NextResponse } from 'next/server';
import { getTransactions } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

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
    return handleApiError(error, 'Tags');
  }
}
