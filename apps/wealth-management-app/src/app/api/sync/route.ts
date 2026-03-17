import { NextResponse } from 'next/server';
import { invalidateCache } from "@wealth-management/utils";

export async function POST() {
  try {
    // Clear all google sheets related caches
    await invalidateCache('accounts');
    await invalidateCache('transactions');
    await invalidateCache('budget');

    return NextResponse.json({ success: true, message: 'Cache cleared successfully' });
  } catch (error) {
    console.error("Sync API Error:", error);
    return NextResponse.json({ error: 'Failed to clear cache' }, { status: 500 });
  }
}
