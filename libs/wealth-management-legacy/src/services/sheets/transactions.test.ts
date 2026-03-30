import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as TransactionsAPI from './transactions';
import { readSheet, appendRow, writeToFirstEmptyRow } from './client';
import { getCached, setCache, invalidateCache } from '@wealth-management/utils';
vi.mock('../db/cache', () => ({
  getCached: vi.fn(),
  setCache: vi.fn(),
  invalidateCache: vi.fn(),
}));

vi.mock('./client', () => ({
  readSheet: vi.fn(),
  appendRow: vi.fn(),
  writeToFirstEmptyRow: vi.fn(),
}));

describe('Google Sheets Transactions Sync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('fetchTransactions', () => {
    it('returns cached data if valid', async () => {
      (getCached as any).mockResolvedValue([{ id: 'row-1', payee: 'Test Payee', payment: 50 }]);

      const txs = await TransactionsAPI.getTransactions();
      expect(txs).toHaveLength(1);
      expect(txs[0].payee).toBe('Test Payee');
      expect(readSheet).not.toHaveBeenCalled();
    });

    it('fetches fresh data from client if cache missed', async () => {
      (getCached as any).mockResolvedValue(null);

      const mockRows = [
        ['Checking', '45658', '', 'Grocery', '', '', 'Food', 'Y', '50000', '', '500000', '500000', '450000'],
      ];

      (readSheet as any).mockResolvedValue(mockRows);

      const txs = await TransactionsAPI.getTransactions();

      expect(txs).toHaveLength(1);
      expect(txs[0].payee).toBe('Grocery');
      expect(txs[0].payment).toBe(50000); // raw VND
      expect(txs[0].category).toBe('Food');
      expect(txs[0].cleared).toBe(true);

      expect(setCache).toHaveBeenCalled();
      expect(readSheet).toHaveBeenCalledWith('Transactions!A2:M');
    });
  });

  describe('addTransaction', () => {
    it('appends a row to the sheet and invalidates cache', async () => {
      (writeToFirstEmptyRow as any).mockResolvedValue(true);

      const newTx = {
        accountName: 'Checking',
        date: new Date('2026-01-01T00:00:00Z'),
        referenceNumber: null,
        payee: 'Store',
        tags: ['sale'],
        memo: null,
        category: 'Shopping',
        cleared: true,
        payment: 100,
        deposit: null,
      };

      await TransactionsAPI.addTransaction(newTx);

      expect(writeToFirstEmptyRow).toHaveBeenCalled();
      const callArgs = (writeToFirstEmptyRow as any).mock.calls[0];
      expect(callArgs[0]).toBe('Transactions');
      expect(callArgs[1]).toBe('Transactions!A2:A');
      // Verify date formatting and other fields
      expect(callArgs[2][0]).toBe('Checking');
      expect(callArgs[2][3]).toBe('Store');
      expect(callArgs[2][4]).toBe('sale');
      expect(callArgs[2][6]).toBe('Shopping');
      expect(callArgs[2][7]).toBe('✓');
      expect(callArgs[2][8]).toBe(100);

      expect(invalidateCache).toHaveBeenCalledWith(expect.stringContaining('transactions'));
    });

    it('formats amounts as Google Finance formulas for Binance accounts', async () => {
      (writeToFirstEmptyRow as any).mockResolvedValue(true);

      const newTx = {
        accountName: 'Binance Earn',
        date: new Date('2026-02-01T00:00:00Z'),
        referenceNumber: null,
        payee: 'Staking Reward',
        tags: ['crypto'],
        memo: null,
        category: 'Interest',
        cleared: true,
        payment: null,
        deposit: 50,
      };

      await TransactionsAPI.addTransaction(newTx as any);

      expect(writeToFirstEmptyRow).toHaveBeenCalled();
      const callArgs = (writeToFirstEmptyRow as any).mock.calls[0];
      expect(callArgs[0]).toBe('Transactions');
      expect(callArgs[2][0]).toBe('Binance Earn');

      // Verify payment was formatted as empty string
      expect(callArgs[2][8]).toBe('');

      // Verify deposit amount was mapped to the dynamic formula
      expect(callArgs[2][9]).toBe(
        `=50 * IF(INDIRECT("A" & ROW())="Binance Earn"; GOOGLEFINANCE("CURRENCY:USDVND"); 1)`,
      );
    });
  });
});
