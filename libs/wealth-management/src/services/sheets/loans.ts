'use server';

import { readSheet } from './client';
import { Loan } from '@wealth-management/types/loans';
import { getCached, setCache } from '@wealth-management/utils';

const CACHE_KEY = 'loans';
const SHEET_NAME = "'Loan / Dept'";

function num(v: string | undefined): number {
  if (!v || v.trim() === '') return 0;
  const n = parseFloat(String(v).replace(/,/g, ''));
  return isNaN(n) ? 0 : n;
}

export async function getLoans(forceFresh = false): Promise<Loan[]> {
  if (!forceFresh) {
    const cached = await getCached<Loan[]>(CACHE_KEY);
    if (cached) return cached;
  }

  // Row 10 (index 9) is where data starts
  const rows = await readSheet(`${SHEET_NAME}!A10:H50`);

  const loans: Loan[] = rows
    .map((row) => {
      const name = row[0]?.trim() || '';
      if (!name) return null;

      return {
        name,
        monthlyDebt: num(row[1]),
        monthlyPaid: num(row[2]),
        monthlyRemaining: num(row[3]),
        extra: row[4] || '',
        yearlyDebt: num(row[5]),
        yearlyPaid: num(row[6]),
        yearlyRemaining: num(row[7]),
      };
    })
    .filter((l): l is Loan => l !== null);

  await setCache(CACHE_KEY, loans, 300);
  return loans;
}
