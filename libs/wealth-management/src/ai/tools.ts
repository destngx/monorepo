import { z } from 'zod';
import { tool } from 'ai';
import { getAccounts, getTransactions, getBudget, getExchangeRate } from '@wealth-management/services/server';
import { convertUSDTtoVND, formatVND } from '@wealth-management/utils';

export const financialTools = {
  getAccountBalances: tool({
    description: 'Get all account balances including bank accounts (VND) and crypto (USDT)',
    inputSchema: z.object({}),
    execute: async () => {
      const accounts = await getAccounts();
      const rate = await getExchangeRate();
      return {
        accounts: accounts.map((a) => ({
          name: a.name,
          balance: a.balance,
          currency: a.currency,
          type: a.type,
          balanceInVND: a.currency === 'USDT' || a.currency === 'USD' ? convertUSDTtoVND(a.balance, rate) : a.balance,
        })),
        exchangeRate: rate,
      };
    },
  }),

  getRecentTransactions: tool({
    description: 'Get recent transactions optionally filtered by a general limit.',
    inputSchema: z.object({
      limit: z.number().optional().default(20).describe('Max transactions to return (default 20, max 100)'),
    }),
    execute: async ({ limit }) => {
      const allTxns = await getTransactions();
      const sorted = allTxns.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
      return sorted.slice(0, Math.min(limit, 100)).map((t) => ({
        date: t.date,
        payee: t.payee,
        category: t.category,
        account: t.accountName,
        payment: t.payment,
        deposit: t.deposit,
        memo: t.memo,
      }));
    },
  }),

  getBudgetStatus: tool({
    description: 'Get budget status showing limits vs requested actual spending',
    inputSchema: z.object({}),
    execute: async () => {
      const budget = await getBudget();
      return {
        categories: budget.map((b) => ({
          category: b.category,
          monthlyLimit: b.monthlyLimit,
          spent: b.monthlySpent,
          remaining: b.monthlyRemaining,
          status: b.monthlySpent > b.monthlyLimit && b.monthlyLimit > 0 ? 'OVER' : 'OK',
        })),
      };
    },
  }),

  webSearch: tool({
    description:
      'Search the web for real-time market data, financial news, crypto prices, interest rates, geopolitical events, or any external information needed to provide accurate advice. Use this proactively when the user asks about current events, prices, rates, or anything requiring up-to-date information.',
    inputSchema: z.object({
      query: z
        .string()
        .describe(
          'The search query — be specific and include relevant financial terms, asset names, dates, or regions',
        ),
    }),
    execute: async ({ query }) => {
      const { executeSearch } = await import('../services/search-service');
      const { results, error } = await executeSearch(query);
      if (error && (!results || results.length === 0)) {
        return {
          results: [],
          error: `Search failed: ${error}. Inform the user and use your internal knowledge instead.`,
        };
      }
      return {
        results: (results || []).map((r) => ({
          title: r.title,
          summary: r.description,
          url: r.url,
        })),
      };
    },
  }),
};
