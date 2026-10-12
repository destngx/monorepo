// Mirrors of the engine's JSON contracts (apps/wealth-management/domain).

export type AccountType =
  | 'active use'
  | 'rarely use'
  | 'long holding'
  | 'deprecated'
  | 'negative active use'
  | 'bank'
  | 'crypto'
  | 'cash'
  | 'investment';

export const ACCOUNT_TYPES: AccountType[] = [
  'active use',
  'rarely use',
  'long holding',
  'negative active use',
  'deprecated',
  'bank',
  'crypto',
  'cash',
  'investment',
];

export type Currency = 'VND' | 'USD' | 'USDT';
export type CategoryType = 'income' | 'expense' | 'non-budget';

export interface Account {
  name: string;
  dueDate: string | null;
  goalAmount: number | null;
  goalProgress: number | null;
  clearedBalance: number;
  balance: number;
  type: AccountType;
  currency: Currency;
  note: string | null;
}

export interface Transaction {
  id: string;
  accountName: string;
  date: string;
  referenceNumber: string | null;
  payee: string;
  tags: string[];
  memo: string | null;
  category: string;
  categoryType?: CategoryType;
  cleared: boolean;
  payment: number | null;
  deposit: number | null;
  accountBalance: number;
  clearedBalance: number;
  runningBalance: number;
  etag: string;
  issues?: string[];
  /** Day the rules attribute the row to (refund memo month rule); absent on older engines. */
  effectiveDate?: string | null;
}

export interface TransactionInput {
  accountName: string;
  date: string;
  referenceNumber?: string;
  payee: string;
  tags: string[];
  memo?: string;
  category: string;
  cleared: boolean;
  payment?: number;
  deposit?: number;
}

export interface TransactionBatchResult {
  batchId: string;
  created: number;
  failed: number;
  replayed: boolean;
  rows: { index: number; id?: string; error?: string }[];
}

export interface Category {
  name: string;
  type: CategoryType;
}

export interface BudgetItem {
  category: string;
  categoryType?: CategoryType;
  monthlyLimit: number;
  yearlyLimit: number;
  monthlyLimits?: Record<string, number>;
  note?: string;
}

export type BudgetStatus = 'healthy' | 'warning' | 'urgent';

export interface BudgetLine {
  category: string;
  categoryType: CategoryType;
  note?: string;
  limit: number;
  spent: number;
  remaining: number;
  percentUsed: number;
  status: BudgetStatus;
  yearlyLimit: number;
  yearlySpent: number;
  yearlyRemaining: number;
  unbudgeted?: boolean;
  /** Day the remaining limit runs out at the current daily burn; null when it will not this month. */
  burnOutDate?: string | null;
}

/** Six-month budget comparison, "The Wall" (WM-BUD-7). */
export interface BudgetHistory {
  /** Oldest first. */
  months: string[];
  rows: { category: string; values: { month: string; limit: number; spent: number }[] }[];
}

export interface ReallocateInput {
  year: number;
  month: string;
  from: string;
  to: string;
  amount: number;
}

export interface ReallocateResult {
  from: { category: string; limit: number };
  to: { category: string; limit: number };
}

export interface BudgetReport {
  month: string;
  daysInMonth: number;
  daysElapsed: number;
  daysRemaining: number;
  totalLimit: number;
  totalSpent: number;
  totalRemaining: number;
  percentUsed: number;
  dailyBurn: number;
  projectedSpend: number;
  status: BudgetStatus;
  lines: BudgetLine[];
}

export interface Goal {
  id: string;
  name: string;
  type: string;
  emoji: string;
  targetAmount: number;
  currentAmount: number;
  deadline: string;
  status: 'ON_TRACK' | 'AT_RISK' | 'OFF_TRACK';
}

export interface Loan {
  name: string;
  monthlyDebt: number;
  monthlyPaid: number;
  monthlyRemaining: number;
  extra: string;
  yearlyDebt: number;
  yearlyPaid: number;
  yearlyRemaining: number;
}

export interface Holding {
  platform: string;
  symbol: string;
  quantity: number | null;
  price: number | null;
  currency: Currency;
  value: number | null;
  valueSource: string;
}

export interface HoldingSection {
  headers: string[];
  holdings: Record<string, unknown>[];
  items: Holding[];
}

export interface InvestmentAssets {
  crypto: HoldingSection;
  funds: HoldingSection;
}

// Engine valuation (WM-INV-2/3/4, WM-RULE-8): every VND figure comes from the engine.
export interface LivePrice {
  price: number;
  currency: Currency;
  provider: string;
  timestamp?: string;
}

export type PortfolioSection = 'crypto' | 'funds';

export interface ValuedHolding extends Holding {
  section: PortfolioSection;
  livePrice?: LivePrice;
  valueVND: number | null;
  issue?: string;
}

export interface PortfolioTotals {
  vndAccounts: number;
  funds: number;
  vnd: number;
  cryptoAccounts: number;
  cryptoHoldings: number;
  crypto: number;
  total: number;
  unvalued: number;
}

export interface PortfolioValuation {
  holdings: ValuedHolding[];
  rates: ExchangeRate[];
  totals: PortfolioTotals;
  issues: string[];
  generatedAt: string;
}

export interface TrendPoint {
  date: string;
  value: number;
}

export interface BalanceSheet {
  assets: number;
  liabilities: number;
  accountsNet: number;
  loansRemaining: number;
  netWorth: number;
}

export interface CashFlow {
  month: string;
  income: number;
  expense: number;
  cashFlow: number;
  savingsRate: number;
}

export interface DashboardSummary {
  asOf: string;
  balanceSheet: BalanceSheet;
  netGrowthThisMonth: number;
  netWorthChangePercent: number | null;
  netWorthTrend: TrendPoint[];
  thisMonth: CashFlow;
  lastMonth: CashFlow;
  incomeChangePercent: number | null;
  expenseChangePercent: number | null;
  cashFlowHistory: CashFlow[];
  topAccounts: Account[];
}

export interface CashbackRule {
  name: string;
  tag: string;
  rate: number;
  capMonthly: number;
  fallback?: boolean;
}

export interface CardCycle {
  statementDay: number;
  dueDay: number;
  cycleStart: string;
  cycleEnd: string;
  cycleDays: number;
  cycleElapsedDays: number;
  cycleProgress: number;
  nextDueDate: string;
  daysUntilDue: number;
  dueUrgent: boolean;
  minimumPayment: number;
  outstandingAmount: number;
}

export interface CardHistoryMonth {
  month: string;
  spent: number;
  potentialCashback: number;
  actualRefund: number;
  efficiency: number;
  missed: number;
}

export type UtilizationBand = 'good' | 'fair' | 'high';

export interface CardReport {
  name: string;
  bank: string;
  keyword: string;
  accountName: string;
  expiry: string;
  limit: number;
  usage: number;
  utilization: number;
  utilizationBand: UtilizationBand;
  remainingLimit: number;
  spentThisCycle: number;
  estimatedCashback: number;
  refundYtd: number;
  annualFee: number;
  netEarnYtd: number;
  lifetimeFees: number;
  /** Card-level monthly cashback cap; 0 means none (WM-CARD-2). */
  monthlyCap: number;
  transactionCount: number;
  cycle: CardCycle;
  rules: CashbackRule[];
  history: CardHistoryMonth[];
  cycleTransactions: Transaction[];
}

export interface CardAccountGroup {
  name: string;
  shared: boolean;
  limit: number;
  usage: number;
  utilization: number;
  remainingLimit: number;
  estimatedCashback: number;
  refundYtd: number;
  annualFees: number;
  netEarnYtd: number;
  cards: CardReport[];
}

export interface CardBankReport {
  name: string;
  accounts: CardAccountGroup[];
}

export interface HealthComponent {
  key: string;
  label: string;
  value: number;
  target: number;
  unit: string;
  score: number;
  maxScore: number;
  formula: string;
  detail: string;
}

export interface FinancialHealth {
  score: number;
  band: 'red' | 'yellow' | 'green';
  components: HealthComponent[];
  recommendations: { component: string; title: string; action: string }[];
  liquidAssets: number;
  avgMonthlyIncome: number;
  avgMonthlyExpense: number;
  netWorth: number;
  freedomMonths: number | null;
  allocation: Record<string, number>;
  lookbackMonths: number;
  insufficientIncome: boolean;
  /** Net worth projected with WM-RULE-12; absent from engines before WM-HEA-2. */
  projection?: NetWorthProjection[];
}

export interface NetWorthProjection {
  years: number;
  nominal: number;
  /** Inflation-adjusted, in today's VND. */
  real: number;
}

export type DependencyStatus = 'ok' | 'degraded' | 'down' | 'unconfigured';

export interface EngineHealth {
  status: DependencyStatus;
  checks: { name: string; status: DependencyStatus; error?: string; latencyMs?: number }[];
}

/** The connected spreadsheet (WM-SET-2). */
export interface SheetInfo {
  spreadsheetId: string;
  title: string;
  /** Engine-side time of the last successful sync; null until one after a restart. */
  lastSyncAt: string | null;
}

export interface SyncResult {
  success: boolean;
  message: string;
  deleted?: number;
}

export interface Ticker {
  symbol: string;
  type: string;
  provider: string;
  currency?: string;
  price?: number;
  timestamp?: string;
}

export interface ExchangeRate {
  from: string;
  to: string;
  rate: number;
  provider: string;
  timestamp?: string;
}

export interface BankRate {
  bank: string;
  tenor: string;
  rate: number;
}

// AI (§10). Every figure in these payloads that is not prose comes from the engine.

export type AIUseCase = 'insight' | 'suggest' | 'vision' | 'chat';
export const AI_USE_CASES: AIUseCase[] = ['insight', 'suggest', 'vision', 'chat'];

export interface AIStatus {
  configured: boolean;
  provider: string;
  endpoint: string;
  models: Partial<Record<AIUseCase, string>>;
  fallbackModel: string;
  prompts: number;
  missingPrompts: string[];
  promptsError?: string;
  capture: CaptureLimits;
}

export type InsightSeverity = 'info' | 'warning' | 'success' | 'critical';
export type InsightIcon =
  | 'alert'
  | 'trend-up'
  | 'trend-down'
  | 'savings'
  | 'tip'
  | 'target'
  | 'shield'
  | 'chart'
  | 'wallet'
  | 'clock'
  | 'zap'
  | 'star';

export interface InsightSection {
  icon: InsightIcon;
  title: string;
  content: string;
  severity: InsightSeverity;
}

export interface Insight {
  summary: string;
  sections: InsightSection[];
  actionItems: string[];
}

export type ReviewKind = 'account-review' | 'budget-review' | 'loan-review' | 'transaction-review' | 'credit-summary';

export interface BriefingAlert {
  severity: InsightSeverity;
  title: string;
  message: string;
}

export interface BriefingPattern {
  kind: string;
  title: string;
  content: string;
}

export interface Briefing {
  briefing: string;
  alerts: BriefingAlert[];
  patterns: BriefingPattern[];
  facts: {
    netWorth: number;
    assets: number;
    liabilities: number;
    income: number;
    expense: number;
    cashFlow: number;
    savingsRate: number;
  };
}

export interface SuggestedLimit {
  category: string;
  currentLimit: number;
  averageSpend: number;
  suggestedLimit: number;
  rationale: string;
}

export interface AdvisorAlert {
  type: 'overspend' | 'spike' | 'bill' | 'opportunity' | 'goal';
  severity: InsightSeverity;
  title: string;
  message: string;
  category?: string;
}

export interface BudgetAdvice {
  month: string;
  briefing: string;
  suggestedLimits: SuggestedLimit[];
  projection: { date: string; balance: number; note?: string }[];
  alerts: AdvisorAlert[];
  goalImpact: { goal: string; impact: string }[];
  forecastedSavings: number;
  unusualVelocity: { category: string; dailyNow: number; dailyAverage: number; ratio: number }[];
}

export interface CategorySuggestion {
  category: string;
  source: 'history' | 'ai' | 'default';
}

export interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface ChatPageContext {
  path: string;
  title?: string;
  summary?: string;
}

export interface ChatEvent {
  type: 'text' | 'tool' | 'done' | 'error';
  delta?: string;
  tool?: string;
  status?: 'running' | 'ok' | 'error';
  model?: string;
  message?: string;
}

// Smart Capture (§6.2). The model only reads materials; matching, transfers,
// dedup and balances are engine code.

export interface CaptureLimits {
  maxItems: number;
  maxItemBytes: number;
  maxBatchBytes: number;
}

export const CAPTURE_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'] as const;

export interface CaptureMaterialPayload {
  id: string;
  kind: 'image' | 'text';
  name?: string;
  mediaType?: string;
  /** Base64 image bytes. */
  data?: string;
  text?: string;
}

export interface CaptureEvidence {
  materialId: string;
  quote?: string;
}

export interface CaptureDuplicate {
  transactionId: string;
  date: string;
  payee: string;
  amount: number;
  reference?: string;
}

export type CaptureField = 'amount' | 'date' | 'account' | 'category';

export interface CaptureSuggestion {
  id: string;
  selected: boolean;
  transaction: TransactionInput;
  direction: 'in' | 'out';
  amount: number;
  currency: string;
  occurredAt: string;
  confidence: number;
  lowConfidence: CaptureField[];
  accountHint?: string;
  accountSource?: 'alias' | 'digits' | 'name';
  accountCandidates?: string[];
  categorySource?: 'rule' | 'history' | 'ai' | 'transfer' | 'default';
  tagOptions?: string[];
  sourceApp?: string;
  evidence: CaptureEvidence[];
  duplicate?: CaptureDuplicate;
  transferGroup?: string;
  notes?: string[];
}

export interface CaptureBalance {
  account: string;
  currency: string;
  balanceAfter: number;
  asOf: string;
  materialId: string;
  sheetBalance: number;
}

export interface CaptureResult {
  suggestions: CaptureSuggestion[];
  unreadable: { materialId: string; reason: string }[];
  balances: CaptureBalance[];
}

export interface CaptureProgress {
  stage: 'extract' | 'match' | 'categorize';
  materialId?: string;
  status?: 'running' | 'ok' | 'unreadable' | 'error';
  events?: number;
  message?: string;
}

export type CaptureEvent =
  | ({ type: 'progress' } & CaptureProgress)
  | { type: 'result'; result: CaptureResult }
  | { type: 'error'; message: string };

export interface CaptureRule {
  kind: 'alias' | 'payee';
  pattern: string;
  value: string;
  updatedAt?: string;
}

export interface CaptureRulesSaveResult {
  created: number;
  updated: number;
  unchanged: number;
}

// Market intelligence (WM-INV-11..18). Every figure below is computed by the
// engine from candles; nothing here comes from a model.

export type Timeframe = '1h' | '4h' | '1d' | '1w';
export const TIMEFRAMES: Timeframe[] = ['1h', '4h', '1d', '1w'];

export type Trend = 'up' | 'down' | 'sideways';
export type Direction = 'up' | 'down' | 'flat';
export type Momentum = 'fire' | 'stable' | 'sleep';

export interface PulseQuote {
  symbol: string;
  name: string;
  provider?: string;
  /** Providers tried before `provider` and why they failed. */
  fallback?: string;
  currency?: string;
  price: number;
  time?: string;
  changePct: number;
  dayChangePct: number | null;
  weekChangePct: number | null;
  direction: Direction;
  momentum: Momentum;
  candles: number;
  error?: string;
}

export interface CorrelationMatrix {
  symbols: string[];
  /** null where two series share too few bars. */
  values: (number | null)[][];
}

export interface Bollinger {
  upper: number;
  middle: number;
  lower: number;
}

export interface Indicators {
  rsi: number | null;
  ema20: number | null;
  ema50: number | null;
  ema200: number | null;
  atr: number | null;
  bollinger: Bollinger | null;
}

export interface TradeSignal {
  action: string;
  confidence: number;
  reasons: string[];
  entry: number;
  stopLoss?: number;
  takeProfit?: number;
  riskReward?: number;
}

export interface FairValueGap {
  type: 'BULLISH' | 'BEARISH';
  top: number;
  bottom: number;
  time: string;
}

export interface Technicals {
  price: number;
  candles: number;
  indicators: Indicators;
  trend: Trend;
  phase: string;
  signal: TradeSignal;
  supportResistance: { support: number[]; resistance: number[] };
  fairValueGaps: FairValueGap[];
  notes: string[];
}

export interface MarketPulse {
  timeframe: Timeframe;
  assets: PulseQuote[];
  topMovers: PulseQuote[];
  correlation: CorrelationMatrix;
  capitalFlow: { signal: 'DEFENSIVE' | 'RISK-ON' | 'MIXED'; summary: string };
  scenario: { name: string; regime: string; summary: string; action: string };
  index: Technicals | null;
  sentiment: { score: number; label: string } | null;
  notes: string[];
  generatedAt: string;
}

export interface TimeframeView {
  timeframe: Timeframe;
  provider?: string;
  candles: number;
  price: number;
  trend: Trend;
  phase: string;
  rsi: number | null;
  error?: string;
}

export interface Confluence {
  views: TimeframeView[];
  pairs: { higher: Timeframe; lower: Timeframe; status: 'ALIGNED' | 'CONFLICT' | 'MIXED' }[];
  bias: Trend;
  score: number;
}

export type SeasonalityType = 'day' | 'week' | 'month';

export interface SeasonalityBucket {
  type: SeasonalityType;
  key: number;
  label: string;
  rank: number;
  avgReturn: number;
  winRate: number;
  profitFactor: number;
  stdDev: number;
  score: number;
  n: number;
}

export interface RiskSimulation {
  horizonDays: number;
  paths: number;
  p10: number;
  p50: number;
  p90: number;
  probabilityOfLoss: number;
  seed: number;
}

export interface RiskStats {
  from: string;
  to: string;
  returns: number;
  periodReturn: number;
  annualisedReturn: number | null;
  annualisedVolatility: number;
  maxDrawdown: number;
  currentDrawdown: number;
  simulation: RiskSimulation | null;
  /** How the figures are computed; shown so heuristics are labelled (WM-INV-16). */
  method: string;
}

export interface MarketAnalysis {
  symbol: string;
  timeframe: Timeframe;
  provider: string;
  fallback?: string;
  currency?: string;
  technicals: Technicals;
  confluence: Confluence;
  seasonality: SeasonalityBucket[];
  risk: RiskStats | null;
  notes: string[];
  generatedAt: string;
}

export interface SearchResult {
  title: string;
  url: string;
  snippet: string;
  source: string;
}

export interface SearchResponse {
  query: string;
  provider: string;
  results: SearchResult[];
  fallback?: string;
}

export type NewsSentiment = 'Positive' | 'Negative' | 'Neutral';
export type NewsBias = 'BULLISH' | 'BEARISH' | 'NEUTRAL';
export const DEFAULT_NEWS_TOPICS = ['Vietnam market & policy', 'Interest rates & macro', 'Global geopolitics'];

export interface NewsArticle extends SearchResult {
  sentiment: NewsSentiment;
  confidence: number;
  impact: number;
  summary: string;
}

export interface NewsReport {
  topic: string;
  query: string;
  bias: NewsBias;
  counts: { positive: number; negative: number; neutral: number };
  articles: NewsArticle[];
  provider: string;
  fallback?: string;
  generatedAt: string;
}

export type InvestmentAction = 'ALLOCATE' | 'HEDGE' | 'OBSERVE';

export interface InvestmentCommand {
  action: InvestmentAction;
  target: string;
  rationale: string;
}

export type InvestmentPhase = 'context' | 'think-tank' | 'synthesis' | 'actions';

/** The raw inputs the think tank saw, disclosed as-is (WM-INV-18). */
export interface ThinkTankContext {
  valuation?: PortfolioValuation;
  valuationError?: string;
  accounts: unknown[];
  loans: unknown[];
  recentTransactions: unknown[];
  budget: { income: number; expenses: number };
  usdtRate?: ExchangeRate;
  searches: SearchResponse[] | null;
  searchErrors?: string[];
  searchContext: string;
  pulse?: MarketPulse;
  pulseError?: string;
  news?: NewsReport[];
}

export type InvestmentEvent =
  | { type: 'phase'; phase: InvestmentPhase; status: 'running' | 'ok' | 'fallback' | 'error'; message?: string }
  | { type: 'context'; context: ThinkTankContext }
  | { type: 'message'; phase: InvestmentPhase; name: string; content: string; model?: string }
  | { type: 'actions'; phase: InvestmentPhase; name: string; commands: InvestmentCommand[]; model?: string }
  | { type: 'done' }
  | { type: 'error'; message: string };
