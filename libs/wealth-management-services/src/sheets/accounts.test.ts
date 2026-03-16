import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as AccountsAPI from './accounts';
import { readSheet } from './client';
import { getCached, setCache } from '../db/cache';

vi.mock('../db/cache', () => ({
  getCached: vi.fn(),
  setCache: vi.fn(),
}));

vi.mock('./client', () => ({
  readSheet: vi.fn(),
}));

describe('Google Sheets Accounts Sync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('fetchAccounts', () => {
    it('returns valid cache data directly without hitting the API', async () => {
      (getCached as any).mockResolvedValue([{ name: 'Cached Account', balance: 999000 }]);

      const accounts = await AccountsAPI.getAccounts();
      expect(accounts).toHaveLength(1);
      expect(accounts[0].name).toBe('Cached Account');
      expect(accounts[0].balance).toBe(999000);
      expect(readSheet).not.toHaveBeenCalled();
    });

    it('ignores expired cache and fetches fresh data instead', async () => {
      (getCached as any).mockResolvedValue(null);

      const mockRows = [['Vietcombank', '', '', '', '200000', '200000', 'active use', '', '']];
      (readSheet as any).mockResolvedValue(mockRows);

      const accounts = await AccountsAPI.getAccounts();
      expect(accounts).toHaveLength(1);
      expect(accounts[0].name).toBe('Vietcombank');
      expect(accounts[0].balance).toBe(200000); // raw VND value

      expect(readSheet).toHaveBeenCalled();
      expect(setCache).toHaveBeenCalled();
    });

    it('filters out help and header rows from sheet', async () => {
      (getCached as any).mockResolvedValue(null);

      const mockRows = [
        ['HELP', '', '', '', '', '', '', ' See Help'],
        ['You can track any number of real or virtual accounts.'],
        ['ACCOUNTS', 'Date to pay', '', '', '', '', 'Type', 'NOTE'],
        ['Cash', '', '1', '100', '1', '1', 'active use', 'VND', ''],
      ];
      (readSheet as any).mockResolvedValue(mockRows);

      const accounts = await AccountsAPI.getAccounts();
      expect(accounts).toHaveLength(1);
      expect(accounts[0].name).toBe('Cash');
    });

    it('gracefully handles empty spreadsheet responses', async () => {
      (getCached as any).mockResolvedValue(null);
      (readSheet as any).mockResolvedValue([]);

      const accounts = await AccountsAPI.getAccounts();
      expect(accounts).toBeInstanceOf(Array);
      expect(accounts).toHaveLength(0);
    });

    it('bubbles up Google Sheet client errors correctly', async () => {
      (getCached as any).mockResolvedValue(null);
      (readSheet as any).mockRejectedValue(new Error('Network error'));

      await expect(AccountsAPI.getAccounts()).rejects.toThrow('Network error');
    });
  });
});
