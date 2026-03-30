export const CACHE_KEYS = {
  ACCOUNTS: 'accounts:all',
  ACCOUNTS_BY_ID: (id: string) => `accounts:${id}`,

  TRANSACTIONS: 'transactions:all',
  TRANSACTIONS_BY_ID: (id: string) => `transactions:${id}`,
  TRANSACTIONS_USER: (userId: string) => `transactions:user:${userId}`,

  BUDGET: 'budget:all',
  BUDGET_USER: (userId: string) => `budget:user:${userId}`,

  EXCHANGE_RATE: 'exchange-rate',

  PRICES: 'prices:all',
  PRICE_SYMBOL: (symbol: string) => `price:${symbol}`,
  PRICES_BATCH: 'prices:batch',

  INVESTMENTS: 'investments:all',
  INVESTMENTS_USER: (userId: string) => `investments:user:${userId}`,

  AI_BUDGET_ADVISOR: 'ai:budget-advisor',
  AI_INVESTMENT_ANALYSIS: 'ai:investment-analysis',
  AI_CREDIT_SUMMARY: 'ai:credit-summary',
  AI_FINANCIAL_HEALTH: 'ai:financial-health',
} as const;

export const CACHE_TTL = {
  PORTFOLIO_DATA: 5 * 60,
  MARKET_DATA: 1 * 60,
  EXCHANGE_RATES: 10 * 60,
  AI_RESPONSES: 30 * 60,
} as const;
