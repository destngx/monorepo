import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as LoansAPI from './loans';
import { readSheet } from './client';
import { getCached, setCache } from '../db/cache';

vi.mock('../db/cache', () => ({
  getCached: vi.fn(),
  setCache: vi.fn(),
}));

vi.mock('./client', () => ({
  readSheet: vi.fn(),
}));

describe('Google Sheets Loans Sync', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fetches and maps loan data correctly', async () => {
    (getCached as any).mockResolvedValue(null);

    const mockRows = [
      ['Cao Hoàng Nam', '0', '0', '0', '', '600000', '600000', '0'],
    ];
    (readSheet as any).mockResolvedValue(mockRows);

    const loans = await LoansAPI.getLoans();

    expect(loans).toHaveLength(1);
    expect(loans[0].name).toBe('Cao Hoàng Nam');
    expect(loans[0].yearlyDebt).toBe(600000);
    expect(loans[0].yearlyRemaining).toBe(0);
    expect(setCache).toHaveBeenCalled();
  });

  it('filters out empty rows', async () => {
    (getCached as any).mockResolvedValue(null);
    const mockRows = [
      ['Lender A', '100', '50', '50', '', '1000', '500', '500'],
      ['', '0', '0', '0', '', '0', '0', '0'],
    ];
    (readSheet as any).mockResolvedValue(mockRows);

    const loans = await LoansAPI.getLoans();
    expect(loans).toHaveLength(1);
    expect(loans[0].name).toBe('Lender A');
  });
});
