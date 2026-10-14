import { aiRequestDone } from '$lib/ai/activity';
import { readNDJSON } from '$lib/ai/ndjson';
import { ApiError, buildPath, request, requestStream, type RequestOptions } from './client';
import type {
  AIProviderUsage,
  AIStatus,
  Account,
  AccountType,
  BankRates,
  Briefing,
  BudgetAdvice,
  BudgetHistory,
  BudgetItem,
  BudgetReport,
  CaptureEvent,
  CaptureMaterialPayload,
  CaptureRule,
  CaptureRulesSaveResult,
  CardBankReport,
  Category,
  CategorySuggestion,
  ChatEvent,
  ChatMessage,
  ChatPageContext,
  DashboardSummary,
  EngineHealth,
  EngineSettings,
  ExchangeRate,
  FinancialHealth,
  GoogleAuthPurpose,
  GoogleAuthResult,
  GoogleAuthStart,
  Goal,
  Insight,
  InvestmentAssets,
  InvestmentEvent,
  MarketAnalysis,
  MarketPulse,
  NewsReport,
  Loan,
  PortfolioValuation,
  ReallocateInput,
  ReallocateResult,
  SettingsChanges,
  SheetInfo,
  ReviewKind,
  SyncResult,
  Ticker,
  Timeframe,
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

// Model calls: each one reports when it ends, so the quota shown refreshes.
const ai = <T>(path: string, body: Record<string, unknown>, { force, signal }: AIOptions = {}) =>
  request<T>(path, { method: 'POST', body: { ...body, force: force || undefined }, signal }).finally(aiRequestDone);

async function* aiStream<T>(path: string, body: unknown, signal?: AbortSignal): AsyncGenerator<T> {
  try {
    yield* readNDJSON<T>(await requestStream(path, { method: 'POST', body, signal }));
  } finally {
    aiRequestDone();
  }
}

export const api = {
  // 503 still carries the per-dependency report, so it is not an error here.
  health: async (fetch: Fetch = globalThis.fetch): Promise<EngineHealth> => {
    const response = await fetch(buildPath('/health'));
    if (response.ok || response.status === 503) return (await response.json()) as EngineHealth;
    throw new ApiError(response.status, (await response.text()).trim() || response.statusText);
  },
  sync: () => request<SyncResult>('/sync', { method: 'POST' }),
  sheetInfo: (fetch?: Fetch) => request<SheetInfo>('/sheets/info', { fetch }),

  settings: () => request<EngineSettings>('/settings'),
  updateSettings: (values: SettingsChanges) =>
    request<EngineSettings>('/settings', { method: 'PATCH', body: { values } }),
  restartEngine: () => request<{ status: string }>('/settings/restart', { method: 'POST' }),
  startGoogleAuth: (purpose: GoogleAuthPurpose) =>
    request<GoogleAuthStart>('/auth/google/start', { method: 'POST', body: { purpose } }),
  completeGoogleAuth: (redirectedUrl: string) =>
    request<GoogleAuthResult>('/auth/google/exchange', { method: 'POST', body: { redirectedUrl } }),

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
  valuation: (options?: ReadOptions) => read<PortfolioValuation>('/investments/valuation', options),

  budget: (year: number, options?: ReadOptions) => read<BudgetItem[]>('/budget', options, { year }),
  updateBudgetLimit: (input: { year: number; category: string; month?: string; limit: number }) =>
    request<{ success: boolean }>('/budget/limits', { method: 'PATCH', body: input }),
  reallocateBudget: (input: ReallocateInput) =>
    request<ReallocateResult>('/budget/reallocate', { method: 'POST', body: input }),

  dashboard: (options?: ReadOptions) => read<DashboardSummary>('/analytics/dashboard', options),
  budgetReport: (month: string, options?: ReadOptions) => read<BudgetReport>('/analytics/budget', options, { month }),
  budgetHistory: (month: string, months: number, options?: ReadOptions) =>
    read<BudgetHistory>('/analytics/budget/history', options, { month, months }),
  cards: (options?: ReadOptions) => read<CardBankReport[]>('/analytics/cards', options),
  financialHealth: (options?: ReadOptions) => read<FinancialHealth>('/analytics/health', options),
  accountHistories: (days: number, options?: ReadOptions) =>
    read<Record<string, TrendPoint[]>>('/analytics/account-histories', options, { days }),

  ticker: (symbol: string, type: string, fetch?: Fetch) =>
    request<Ticker>('/market/ticker', { fetch, query: { symbol, type } }),
  exchangeRate: (from: string, to: string, fetch?: Fetch) =>
    request<ExchangeRate>('/market/exchange-rate', { fetch, query: { from, to } }),
  bankRates: (fetch?: Fetch) => request<BankRates>('/market/bank-rates', { fetch }),
  marketPulse: (timeframe: Timeframe, options?: ReadOptions) =>
    read<MarketPulse>('/market/pulse', options, { timeframe }),
  marketAnalysis: (symbol: string, timeframe: Timeframe, options?: ReadOptions) =>
    read<MarketAnalysis>('/market/analysis', options, { symbol, timeframe }),
  /** Raw fmarket explorer actions (WM-MKT-6); the payload is fmarket's own JSON. */
  fmarket: <T>(
    action: string,
    params: Record<string, unknown> = {},
    options: ReadOptions & { signal?: AbortSignal } = {},
  ) =>
    request<T>('/fmarket/action', {
      method: 'POST',
      body: { action, params },
      fetch: options.fetch,
      signal: options.signal,
      query: { force: options.force || undefined },
    }),

  aiStatus: (fetch?: Fetch) => request<AIStatus>('/ai/status', { fetch }),
  aiUsage: (fetch?: Fetch) => request<AIProviderUsage>('/ai/usage', { fetch }),
  aiModels: () => request<{ models: string[] }>('/ai/models'),
  suggestCategory: (payee: string, signal?: AbortSignal) =>
    request<CategorySuggestion>('/ai/suggest-category', {
      method: 'POST',
      body: { payee },
      signal,
    }).finally(aiRequestDone),
  review: (kind: ReviewKind, input: { month?: string; card?: string } = {}, options?: AIOptions) =>
    ai<Insight>('/ai/review', { kind, ...input }, options),
  healthCommentary: (options?: AIOptions) => ai<Insight>('/ai/health-commentary', {}, options),
  briefing: (options?: AIOptions) => ai<Briefing>('/ai/briefing', {}, options),
  budgetAdvice: (options?: AIOptions) => ai<BudgetAdvice>('/ai/budget-advisor', {}, options),
  chartInsight: (
    input: { chartType: string; title: string; description?: string; timeframe?: string; data: unknown },
    options?: AIOptions,
  ) => ai<{ insight: string }>('/ai/chart-insight', input, options).then((result) => result.insight),
  newsSentiment: (topic: string, options?: AIOptions) => ai<NewsReport>('/ai/news-sentiment', { topic }, options),
  tickerInsight: (symbol: string, timeframe: Timeframe, options?: AIOptions) =>
    ai<Insight>('/ai/ticker-insight', { symbol, timeframe }, options),
  /** Streams the think tank (WM-AI-9); failures after the stream starts arrive as an "error" event. */
  investmentAnalysis: (news: NewsReport[], signal?: AbortSignal) =>
    aiStream<InvestmentEvent>('/ai/investment-analysis', { news: news.length ? news : undefined }, signal),
  /** Streams chat events (WM-AI-11); errors after the stream starts arrive as an "error" event. */
  chat: (messages: ChatMessage[], page: ChatPageContext | undefined, signal?: AbortSignal) =>
    aiStream<ChatEvent>('/ai/chat', { messages, page }, signal),
  /** Streams Smart Capture progress then the result (§6.2); failures after the stream starts arrive as an "error" event. */
  capture: (materials: CaptureMaterialPayload[], signal?: AbortSignal) =>
    aiStream<CaptureEvent>('/ai/capture', { materials }, signal),
  captureRules: (options?: ReadOptions) => read<CaptureRule[]>('/capture/rules', options),
  saveCaptureRules: (rules: CaptureRule[]) =>
    request<CaptureRulesSaveResult>('/capture/rules', { method: 'POST', body: { rules } }),
};
