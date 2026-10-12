import { readNDJSON } from '$lib/ai/ndjson';
import { aiModels } from '$lib/state/settings.svelte';
import { ApiError, buildPath, request, requestStream, type RequestOptions } from './client';
import type {
  AIStatus,
  Account,
  AccountType,
  BankRate,
  Briefing,
  BudgetAdvice,
  BudgetItem,
  BudgetReport,
  CardBankReport,
  Category,
  CategorySuggestion,
  ChatEvent,
  ChatMessage,
  ChatPageContext,
  DashboardSummary,
  EngineHealth,
  ExchangeRate,
  FinancialHealth,
  Goal,
  Insight,
  InvestmentAssets,
  Loan,
  ReviewKind,
  SyncResult,
  Ticker,
  TransactionBatchResult,
  Transaction,
  TransactionInput,
  TrendPoint,
} from './types';

type Fetch = RequestOptions['fetch'];

// Read options shared by every list call: `force` bypasses the cache.
interface ReadOptions {
  fetch?: Fetch;
  force?: boolean;
}

const read = <T>(path: string, { fetch, force }: ReadOptions = {}, query: RequestOptions['query'] = {}) =>
  request<T>(path, { fetch, query: { ...query, force: force || undefined } });

// AI options: `force` bypasses the engine's 30-minute result cache.
interface AIOptions {
  force?: boolean;
  signal?: AbortSignal;
}

const insightModel = () => aiModels.insight.value || undefined;

const ai = <T>(path: string, body: Record<string, unknown>, { force, signal }: AIOptions = {}) =>
  request<T>(path, { method: 'POST', body: { ...body, force: force || undefined, model: insightModel() }, signal });

export const api = {
  // 503 still carries the per-dependency report, so it is not an error here.
  health: async (fetch: Fetch = globalThis.fetch): Promise<EngineHealth> => {
    const response = await fetch(buildPath('/health'));
    if (response.ok || response.status === 503) return (await response.json()) as EngineHealth;
    throw new ApiError(response.status, (await response.text()).trim() || response.statusText);
  },
  sync: () => request<SyncResult>('/sync', { method: 'POST' }),

  accounts: (options?: ReadOptions) => read<Account[]>('/accounts', options),
  createAccount: (input: { name: string; type: AccountType; balance?: number; currency?: string; note?: string }) =>
    request<{ success: boolean }>('/accounts', { method: 'POST', body: input }),
  updateAccount: (name: string, input: { type?: AccountType; note?: string }) =>
    request<Account>(`/accounts/${encodeURIComponent(name)}`, { method: 'PATCH', body: input }),
  deleteAccount: (name: string) =>
    request<{ success: boolean }>(`/accounts/${encodeURIComponent(name)}`, { method: 'DELETE' }),

  transactions: (options?: ReadOptions) => read<Transaction[]>('/transactions', options),
  createTransaction: (input: TransactionInput) =>
    request<{ id: string }>('/transactions', { method: 'POST', body: input }),
  createTransactions: (batchId: string, transactions: TransactionInput[]) =>
    request<TransactionBatchResult>('/transactions/batch', { method: 'POST', body: { batchId, transactions } }),
  updateTransaction: (id: string, etag: string, input: TransactionInput) =>
    request<{ id: string }>(`/transactions/${encodeURIComponent(id)}`, {
      method: 'PATCH',
      body: { ...input, etag },
    }),
  deleteTransaction: (id: string, etag: string) =>
    request<{ success: boolean }>(`/transactions/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: { 'If-Match': etag },
    }),

  categories: (options?: ReadOptions) => read<Category[]>('/categories', options),
  tags: (options?: ReadOptions) => read<string[]>('/tags', options),
  goals: (options?: ReadOptions) => read<Goal[]>('/goals', options),
  loans: (options?: ReadOptions) => read<Loan[]>('/loans', options),
  investments: (options?: ReadOptions) => read<InvestmentAssets>('/investments/assets', options),

  budget: (year: number, options?: ReadOptions) => read<BudgetItem[]>('/budget', options, { year }),
  updateBudgetLimit: (input: { year: number; category: string; month?: string; limit: number }) =>
    request<{ success: boolean }>('/budget/limits', { method: 'PATCH', body: input }),

  dashboard: (options?: ReadOptions) => read<DashboardSummary>('/analytics/dashboard', options),
  budgetReport: (month: string, options?: ReadOptions) => read<BudgetReport>('/analytics/budget', options, { month }),
  cards: (options?: ReadOptions) => read<CardBankReport[]>('/analytics/cards', options),
  financialHealth: (options?: ReadOptions) => read<FinancialHealth>('/analytics/health', options),
  accountHistories: (days: number, options?: ReadOptions) =>
    read<Record<string, TrendPoint[]>>('/analytics/account-histories', options, { days }),

  ticker: (symbol: string, type: string, fetch?: Fetch) =>
    request<Ticker>('/market/ticker', { fetch, query: { symbol, type } }),
  exchangeRate: (from: string, to: string, fetch?: Fetch) =>
    request<ExchangeRate>('/market/exchange-rate', { fetch, query: { from, to } }),
  bankRates: (fetch?: Fetch) => request<BankRate[]>('/market/bank-rates', { fetch }),

  aiStatus: (fetch?: Fetch) => request<AIStatus>('/ai/status', { fetch }),
  suggestCategory: (payee: string, signal?: AbortSignal) =>
    request<CategorySuggestion>('/ai/suggest-category', {
      method: 'POST',
      body: { payee, model: aiModels.suggest.value || undefined },
      signal,
    }),
  review: (kind: ReviewKind, input: { month?: string; card?: string } = {}, options?: AIOptions) =>
    ai<Insight>('/ai/review', { kind, ...input }, options),
  healthCommentary: (options?: AIOptions) => ai<Insight>('/ai/health-commentary', {}, options),
  briefing: (options?: AIOptions) => ai<Briefing>('/ai/briefing', {}, options),
  budgetAdvice: (options?: AIOptions) => ai<BudgetAdvice>('/ai/budget-advisor', {}, options),
  chartInsight: (
    input: { chartType: string; title: string; description?: string; timeframe?: string; data: unknown },
    options?: AIOptions,
  ) => ai<{ insight: string }>('/ai/chart-insight', input, options).then((result) => result.insight),
  /** Streams chat events (WM-AI-11); errors after the stream starts arrive as an "error" event. */
  chat: async function* (
    messages: ChatMessage[],
    page: ChatPageContext | undefined,
    signal?: AbortSignal,
  ): AsyncGenerator<ChatEvent> {
    const stream = await requestStream('/ai/chat', {
      method: 'POST',
      body: { messages, page, model: aiModels.chat.value || undefined },
      signal,
    });
    yield* readNDJSON<ChatEvent>(stream);
  },
};
