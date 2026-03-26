'use server';

import { readSheet, writeToFirstEmptyRow } from './client';
import { mapTransaction } from './mappers';
import { Transaction } from '@wealth-management/types/transactions';
import { formatDateForSheets } from '@wealth-management/utils';

export async function getTransactions(): Promise<Transaction[]> {
  // Row 1 is header
  const rows = await readSheet('Transactions!A2:M');
  return rows
    .map((row, i) => mapTransaction(row, i + 2))
    .filter((t): t is Transaction => t !== null && !!(t.payee || t.payment || t.deposit));
}

export async function addTransaction(
  data: Omit<Transaction, 'id' | 'accountBalance' | 'clearedBalance' | 'runningBalance'>,
) {
  const formatAmount = (amount: number | null, accountName: string) => {
    if (amount === null) return '';
    if (accountName.toLowerCase().includes('binance')) {
      return `=${amount} * IF(INDIRECT("A" & ROW())="${accountName}"; GOOGLEFINANCE("CURRENCY:USDVND"); 1)`;
    }
    return amount;
  };

  const rowData = [
    data.accountName,
    formatDateForSheets(data.date),
    data.referenceNumber || '',
    data.payee,
    data.tags.join(','),
    data.memo || '',
    data.category,
    data.cleared ? '✓' : '',
    formatAmount(data.payment, data.accountName),
    formatAmount(data.deposit, data.accountName),
  ];

  // Write to the first empty row inside the pre-defined table range.
  // Column A (accountName) is used as the anchor to find where real data ends.
  await writeToFirstEmptyRow('Transactions', 'Transactions!A2:A', rowData);
}
