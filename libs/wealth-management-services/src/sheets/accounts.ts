import { readSheet } from './client';
import { mapAccount } from './mappers';
import { Account } from '../types/account';
import { getCached, setCache } from '../db/cache';

const CACHE_KEY = 'accounts';

export async function getAccounts(forceFresh = false): Promise<Account[]> {
  if (!forceFresh) {
    const cached = await getCached<Account[]>(CACHE_KEY);
    if (cached) return cached;
  }

  // Assuming Accounts is on a sheet named "Accounts", headers on row 1
  const rows = await readSheet('Accounts!A2:I');

  const accounts = rows.map(mapAccount).filter((a): a is Account => a !== null && a.name !== '');
  await setCache(CACHE_KEY, accounts, 300); // 5 min cache
  return accounts;
}
