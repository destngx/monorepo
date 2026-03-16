import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as BudgetAPI from './budget';
import { readSheet } from './client';
import { getCached, setCache } from '../db/cache';

vi.mock('../db/cache', () => ({
  getCached: vi.fn(),
  setCache: vi.fn(),
}));

vi.mock('./client', () => ({
  readSheet: vi.fn(),
}));

describe('Google Sheets Budget Sync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('fetchBudget', () => {
    it('returns cached data if valid', async () => {
      (getCached as any).mockResolvedValue([{ category: 'Food', monthlyLimit: 500 }]);

      const budget = await BudgetAPI.getBudget();
      expect(budget).toHaveLength(1);
      expect(budget[0].category).toBe('Food');
      expect(readSheet).not.toHaveBeenCalled();
    });

    it('fetches fresh data from client if cache missed', async () => {
      (getCached as any).mockResolvedValue(null);

      // The budget parser expects headers at row 10 (index 9) and data from row 12+ (index 11)
      const mockRows = new Array(12).fill([]);
      mockRows[9] = ['', '', '', '45600', '45631']; // Month serials (Jan, Feb)
      mockRows[11] = ['Food', '', '6000000', '500000', '500000']; // Category, Yearly, Jan, Feb

      (readSheet as any).mockResolvedValue(mockRows);

      const budget = await BudgetAPI.getBudget();

      expect(budget).toHaveLength(1);
      expect(budget[0].category).toBe('Food');
      expect(budget[0].yearlyLimit).toBe(6000000);
      expect(budget[0].monthlyLimit).toBe(500000); // 6,000,000 / 12

      expect(setCache).toHaveBeenCalled();
      expect(readSheet).toHaveBeenCalledWith('Budget_2026!A1:N200');
    });
  });
});
