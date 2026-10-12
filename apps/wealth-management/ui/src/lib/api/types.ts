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
}

export type DependencyStatus = 'ok' | 'degraded' | 'down' | 'unconfigured';

export interface EngineHealth {
  status: DependencyStatus;
  checks: { name: string; status: DependencyStatus; error?: string; latencyMs?: number }[];
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
