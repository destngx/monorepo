import { aiRequestDone } from '$lib/ai/activity';
import { readNDJSON } from '$lib/ai/ndjson';
import { ApiError, buildPath, request, requestStream, type RequestOptions } from './client';
import type {
  AccountReadinessInput,
  ApiTokenStatus,
  EmergencyAccountInput,
  AccountMinimumInput,
  AIProviderUsage,
  Allocation,
  AIStatus,
  Account,
  AccountType,
  BankRates,
  Bills,
  BillsSummary,
  MonthlyClose,
  NetWorthHistory,
  SnapshotSaveResult,
  IdleCash,
  Briefing,
  BudgetAdvice,
  BudgetHistory,
  BudgetItem,
  BudgetReport,
  CaptureDraftEdit,
  CaptureDrafts,
  CaptureEvent,
  CaptureMaterialPayload,
  CaptureRule,
  CaptureRulesSaveResult,
  CardBankReport,
  CardConfig,
  CardPaymentQuery,
  CardSuggestion,
  Category,
  CategoryDefinitionInput,
  CategoryNatureInput,
  CategorySuggestion,
  ChatEvent,
  ChatMessage,
  ChatPageContext,
  DashboardSummary,
  EngineHealth,
  EngineSettings,
  ExchangeRate,
  Exposure,
  FinancialHealth,
  FundPerformance,
  GoogleAuthPurpose,
  GoogleAuthResult,
  GoogleAuthStart,
  GoalInput,
  GoalPlan,
  GoalPlanSetupResult,
  Hedges,
  PlannedGoal,
  Insight,
  InvestmentAssets,
  InvestmentEvent,
  MarketAnalysis,
  MarketPulse,
  MoneyLent,
  NeedsYou,
  NewsReport,
  PortfolioValuation,
  PreferenceValue,
  Preferences,
  QuickCaptureOverview,
  QuickCaptureReceipt,
  ProgramComparisonReport,
  Recommendation,
  RecommendationFilter,
  RecommendationList,
  RecommendationStatusInput,
  ReallocateInput,
  ReallocateResult,
  RecurringConfirmInput,
  RecurringInput,
  RecurringItem,
  RecurringStatus,
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
  YearlyCashFlow,
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
  apiTokenStatus: () => request<ApiTokenStatus>('/settings/api-token'),
  /** Applies at once: later writes need the new token. */
  setApiToken: (token: string) => request<ApiTokenStatus>('/settings/api-token', { method: 'POST', body: { token } }),
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
  goalPlan: (options?: ReadOptions) => read<GoalPlan>('/goals', options),
  setupGoalPlan: () => request<GoalPlanSetupResult>('/goals/setup', { method: 'POST' }),
  createGoal: (input: GoalInput) => request<PlannedGoal>('/goals', { method: 'POST', body: input }),
  updateGoal: (id: string, etag: string, input: GoalInput) =>
    request<PlannedGoal>(`/goals/${encodeURIComponent(id)}`, { method: 'PATCH', body: { ...input, etag } }),
  deleteGoal: (id: string, etag: string) =>
    request<{ success: boolean }>(`/goals/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: { 'If-Match': etag },
    }),
  reorderGoals: (ids: string[]) => request<PlannedGoal[]>('/goals/reorder', { method: 'POST', body: { ids } }),
  setCategoryNature: (input: CategoryNatureInput) =>
    request<Category>('/categories/nature', { method: 'PATCH', body: input }),
  /** Definition and split tags, Categories columns D and E (WM-CAT-1, WM-CAT-2). */
  setCategoryDefinition: (input: CategoryDefinitionInput) =>
    request<Category>('/categories/definition', { method: 'PATCH', body: input }),
  moneyLent: (options?: ReadOptions) => read<MoneyLent>('/lending', options),
  /** Bills, suggestions, the 30-day calendar and safe-to-spend (ROADMAP R4). */
  bills: (options?: ReadOptions) => read<Bills>('/bills', options),
  safeToSpend: (options?: ReadOptions) => read<BillsSummary>('/bills/safe-to-spend', options),
  /** Idle cash against the best instant-access deposit rate (ROADMAP R6, D34). */
  idleCash: (options?: ReadOptions) => read<IdleCash>('/bills/idle-cash', options),
  monthlyClose: (month?: string, options?: ReadOptions) =>
    read<MonthlyClose>('/close', options, { month: month || undefined }),
  writeSnapshot: (month: string) => request<SnapshotSaveResult>('/close/snapshot', { method: 'POST', body: { month } }),
  netWorthHistory: (options?: ReadOptions) => read<NetWorthHistory>('/net-worth/history', options),
  /** Marks an account ready to use or not for safe to spend (D33); 409 when the setting changed. */
  setAccountReady: (input: AccountReadinessInput) =>
    request<Preferences>('/bills/safe-to-spend/accounts', { method: 'POST', body: input }),
  /** Marks an account as an emergency fund account or not for idle cash (D34); 409 when the setting changed. */
  setEmergencyAccount: (input: EmergencyAccountInput) =>
    request<Preferences>('/bills/idle-cash/emergency-accounts', { method: 'POST', body: input }),
  /** Sets or removes (null or 0) an account's minimum balance to keep (D35); 409 when the setting changed. */
  setAccountMinimum: (input: AccountMinimumInput) =>
    request<Preferences>('/bills/safe-to-spend/minimum-balance', { method: 'POST', body: input }),
  createRecurring: (input: RecurringInput) => request<RecurringItem>('/bills/items', { method: 'POST', body: input }),
  confirmSuggestion: (input: RecurringConfirmInput) =>
    request<RecurringItem>('/bills/suggestions/confirm', { method: 'POST', body: input }),
  updateRecurring: (id: string, etag: string, input: RecurringInput) =>
    request<RecurringItem>(`/bills/items/${encodeURIComponent(id)}`, { method: 'PATCH', body: { ...input, etag } }),
  setRecurringStatus: (id: string, etag: string, status: RecurringStatus) =>
    request<RecurringItem>(`/bills/items/${encodeURIComponent(id)}/status`, { method: 'POST', body: { status, etag } }),
  deleteRecurring: (id: string, etag: string) =>
    request<{ success: boolean }>(`/bills/items/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: { 'If-Match': etag },
    }),
  /** The recommendations list as stored; reads only (ROADMAP R7). */
  recommendations: (filter: RecommendationFilter = {}, options?: ReadOptions) =>
    read<RecommendationList>('/recommendations', options, { ...filter }),
  /** Runs the engine rules, then lists; writes the Recommendations tab. */
  refreshRecommendations: (filter: RecommendationFilter = {}, fetch?: Fetch) =>
    request<RecommendationList>('/recommendations/refresh', { method: 'POST', fetch, query: { ...filter } }),
  setRecommendationStatus: (id: string, etag: string, input: RecommendationStatusInput) =>
    request<Recommendation>(`/recommendations/${encodeURIComponent(id)}/status`, {
      method: 'POST',
      body: { ...input, etag },
    }),
  investments: (options?: ReadOptions) => read<InvestmentAssets>('/investments/assets', options),
  valuation: (options?: ReadOptions) => read<PortfolioValuation>('/investments/valuation', options),
  /** The one allocation with targets and drift (ROADMAP C4). */
  allocation: (options?: ReadOptions) => read<Allocation>('/investments/allocation', options),
  /** Fund holdings vs a bank deposit (WM-INV-21/22). */
  fundPerformance: (options?: ReadOptions) => read<FundPerformance>('/investments/fund-performance', options),
  /** My exposure: volatile share, today's change and the crypto view (ROADMAP M1/M2). */
  exposure: (options?: ReadOptions) => read<Exposure>('/investments/exposure', options),
  /** Gold and USD/VND trends against the VND cash share (ROADMAP M4). */
  hedges: (options?: ReadOptions) => read<Hedges>('/investments/hedges', options),
  /** The app-managed Preferences tab (WM-SET-6). */
  preferences: (options?: ReadOptions) => read<Preferences>('/preferences', options),
  savePreferences: (values: PreferenceValue[]) =>
    request<Preferences>('/preferences', { method: 'POST', body: { values } }),

  budget: (year: number, options?: ReadOptions) => read<BudgetItem[]>('/budget', options, { year }),
  updateBudgetLimit: (input: { year: number; category: string; month?: string; limit: number }) =>
    request<{ success: boolean }>('/budget/limits', { method: 'PATCH', body: input }),
  reallocateBudget: (input: ReallocateInput) =>
    request<ReallocateResult>('/budget/reallocate', { method: 'POST', body: input }),

  dashboard: (options?: ReadOptions) => read<DashboardSummary>('/analytics/dashboard', options),
  /** The home inbox, ranked (ROADMAP R5). */
  needsYou: (options?: ReadOptions) => read<NeedsYou>('/home/needs-you', options),
  budgetReport: (month: string, options?: ReadOptions) => read<BudgetReport>('/analytics/budget', options, { month }),
  budgetHistory: (month: string, months: number, options?: ReadOptions) =>
    read<BudgetHistory>('/analytics/budget/history', options, { month, months }),
  /** Without a year the engine uses the current one. */
  yearlyCashFlow: (year: number | undefined, options?: ReadOptions) =>
    read<YearlyCashFlow>('/analytics/cash-flow/yearly', options, { year }),
  cards: (options?: ReadOptions) => read<CardBankReport[]>('/analytics/cards', options),
  suggestCards: ({ payee, amount, mcc }: CardPaymentQuery, options?: ReadOptions) =>
    read<CardSuggestion[]>('/analytics/cards/suggest', options, { payee, amount, mcc }),
  comparePrograms: (options?: ReadOptions) => read<ProgramComparisonReport>('/analytics/cards/compare', options),
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
  review: (kind: ReviewKind, input: { month?: string; card?: string; year?: number } = {}, options?: AIOptions) =>
    ai<Insight>('/ai/review', { kind, ...input }, options),
  healthCommentary: (options?: AIOptions) => ai<Insight>('/ai/health-commentary', {}, options),
  briefing: (options?: AIOptions) => ai<Briefing>('/ai/briefing', {}, options),
  budgetAdvice: (options?: AIOptions) => ai<BudgetAdvice>('/ai/budget-advisor', {}, options),
  chartInsight: (
    input: { chartType: string; title: string; description?: string; timeframe?: string; data: unknown },
    options?: AIOptions,
  ) => ai<{ insight: string }>('/ai/chart-insight', input, options).then((result) => result.insight),
  explainCashback: (input: { card: string; month?: string }, options?: AIOptions) =>
    ai<{ explanation: string }>('/ai/cashback-explain', input, options).then((result) => result.explanation),
  newsSentiment: (topic: string, options?: AIOptions) => ai<NewsReport>('/ai/news-sentiment', { topic }, options),
  tickerInsight: (symbol: string, timeframe: Timeframe, options?: AIOptions) =>
    ai<Insight>('/ai/ticker-insight', { symbol, timeframe }, options),
  /**
   * Streams the think tank (WM-AI-9); failures after the stream starts arrive as an "error" event.
   * No symbols means the user's current holdings (ROADMAP M5).
   */
  investmentAnalysis: (news: NewsReport[], symbols: string[], signal?: AbortSignal) =>
    aiStream<InvestmentEvent>(
      '/ai/investment-analysis',
      { news: news.length ? news : undefined, symbols: symbols.length ? symbols : undefined },
      signal,
    ),
  /** Streams chat events (WM-AI-11); errors after the stream starts arrive as an "error" event. */
  chat: (messages: ChatMessage[], page: ChatPageContext | undefined, signal?: AbortSignal) =>
    aiStream<ChatEvent>('/ai/chat', { messages, page }, signal),
  /** Streams Smart Capture progress then the result (§6.2); failures after the stream starts arrive as an "error" event. */
  capture: (materials: CaptureMaterialPayload[], signal?: AbortSignal) =>
    aiStream<CaptureEvent>('/ai/capture', { materials }, signal),
  /** The sheet's card configuration: programmes, banks, cards held and merchant MCCs. */
  cardPrograms: (options?: ReadOptions) => read<CardConfig>('/cards/programs', options),
  captureRules: (options?: ReadOptions) => read<CaptureRule[]>('/capture/rules', options),
  saveCaptureRules: (rules: CaptureRule[]) =>
    request<CaptureRulesSaveResult>('/capture/rules', { method: 'POST', body: { rules } }),

  /** Quick capture status and the number of drafts waiting for review (WM-CAP-19). */
  captureInbox: (options?: ReadOptions) => read<QuickCaptureOverview>('/capture/inbox', options),
  /** Sends what the iOS Shortcut sends: an image and/or a note as a multipart form (WM-CAP-15). */
  submitCapture: (image: Blob | undefined, note: string) => {
    const form = new FormData();
    if (image) form.append('image', image, 'capture');
    if (note.trim()) form.append('note', note.trim());
    return request<QuickCaptureReceipt>('/capture/inbox', { method: 'POST', body: form });
  },
  retryCapture: (id: string) =>
    request<{ status: string }>(`/capture/inbox/${encodeURIComponent(id)}/retry`, { method: 'POST' }),
  discardCapture: (id: string) => request<void>(`/capture/inbox/${encodeURIComponent(id)}`, { method: 'DELETE' }),
  captureDrafts: (options?: ReadOptions) => read<CaptureDrafts>('/capture/drafts', options),
  /** Writes the edited drafts through the batch path; replaying batchId does not write twice. */
  approveDrafts: (batchId: string, drafts: { id: string; transaction: TransactionInput }[]) =>
    request<TransactionBatchResult>('/capture/drafts/approve', { method: 'POST', body: { batchId, drafts } }),
  editDraft: (id: string, edit: CaptureDraftEdit) =>
    request<void>(`/capture/drafts/${encodeURIComponent(id)}`, { method: 'PATCH', body: edit }),
  rejectDrafts: (ids: string[]) => request<void>('/capture/drafts/reject', { method: 'POST', body: { ids } }),
  captureMaterialUrl: (id: string) => buildPath(`/capture/materials/${encodeURIComponent(id)}`),
};
