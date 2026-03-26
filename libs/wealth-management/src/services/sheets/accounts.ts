import { readSheet } from './client';
import { mapAccount } from './mappers';
import { Account } from '@wealth-management/types/accounts';

export async function getAccounts(): Promise<Account[]> {
  // Assuming Accounts is on a sheet named "Accounts", headers on row 1
  const rows = await readSheet('Accounts!A2:I');

  return rows.map(mapAccount).filter((a): a is Account => a !== null && a.name !== '');
}
