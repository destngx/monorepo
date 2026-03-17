import { describe, it, expect } from 'vitest';
import { buildSystemPrompt } from './system-prompt';
import * as AccountsAPI from '@wealth-management/services';
import * as BudgetAPI from '@wealth-management/services';
import { vi } from 'vitest';

vi.mock('../sheets/accounts', () => ({
  getAccounts: vi.fn(),
}));

vi.mock('../sheets/budget', () => ({
  getBudget: vi.fn(),
}));

vi.mock('../services/exchange-rate-service', () => ({
  getExchangeRate: vi.fn().mockResolvedValue(25000),
}));

describe('AI System Prompt Builder', () => {
  it('builds system prompt correctly injecting context from APIs', async () => {
    (AccountsAPI.getAccounts as any).mockResolvedValue([
      { name: 'Checking', type: 'bank', balance: 100000, currency: 'VND' },
      { name: 'Savings', type: 'bank', balance: 50, currency: 'USD' }, // 50 * 25k (default fallback rate) = 1,250,000 VND
    ]);
    (BudgetAPI.getBudget as any).mockResolvedValue([
      { category: 'Food', monthlyLimit: 1000, monthlySpent: 200 },
      { category: 'Unused', monthlyLimit: 0, monthlySpent: 0 }, // should be filtered out
    ]);

    const prompt = await buildSystemPrompt();

    expect(prompt).toContain('personal wealth management advisor');

    // 100000 + 1,250,000 = 1,350,000 Total Net worth
    expect(prompt).toContain('1.350.000');

    expect(prompt).toContain('Checking: 100.000');
    expect(prompt).toContain('Savings: 50 USD');

    // Should filter limit 0
    expect(prompt).toContain('Food: 200');
    expect(prompt).not.toContain('Unused:');
  });

  it('handles empty states gracefully when APIs reject with errors', async () => {
    (AccountsAPI.getAccounts as any).mockRejectedValue(new Error('Google API Failed'));
    (BudgetAPI.getBudget as any).mockRejectedValue(new Error('Drive down'));

    const prompt = await buildSystemPrompt();

    expect(prompt).toContain('Total Net Worth: 0');
    expect(prompt).toContain('No accounts synced.');
    expect(prompt).toContain('No budget info set.');
  });
});
