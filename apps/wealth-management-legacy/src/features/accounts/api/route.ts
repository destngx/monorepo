import { NextResponse } from 'next/server';
import { getAccounts } from '@wealth-management/services/server';
import { handleApiError } from '@wealth-management/utils/server';
import { getOrSetCache, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const forceFresh = searchParams.get('force') === 'true';

    const accounts = await getAccounts(forceFresh);

    return NextResponse.json(accounts);
  } catch (error) {
    return handleApiError(error, 'Accounts');
  }
}
