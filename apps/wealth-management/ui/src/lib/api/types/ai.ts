import type { CategoryNature } from './budget';
import type { CaptureLimits } from './capture';
import type { FillChatForm } from './fill';
import type { RecommendationRecord } from './recommendations';

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

/** OpenAI quota from the ai-gateway (WM-SET-4). */
export interface AIUsageWindow {
  /** "5h" or "weekly". */
  name: string;
  usedPercent: number;
  resetsAt?: string;
}

export interface AIUsageTotals {
  since?: string;
  inputTokens: number;
  outputTokens: number;
  requests: number;
}

export interface AIProviderUsage {
  provider: string;
  authMode?: string;
  source?: string;
  /** Rate-limit windows, reported for OAuth accounts. */
  windows: AIUsageWindow[];
  /** Token totals, reported for API-key accounts. */
  totals?: AIUsageTotals;
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
  /** What a review wrote to the recommendations list (WM-REC-2). */
  recorded?: RecommendationRecord;
}

export type ReviewKind = 'account-review' | 'transaction-review' | 'credit-summary' | 'year-review';

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
  /** What the briefing wrote to the recommendations list (WM-REC-2). */
  recorded?: RecommendationRecord;
}

/** Last 3 full months against earlier this year (WM-BUD-9). */
export type SpendTrend = 'rising' | 'falling' | 'steady' | 'unknown';
/** How a category's spend is spread over its months (WM-BUD-9). */
export type SpendPattern = 'steady' | 'variable' | 'occasional';
/** Which way a recommendation moves a limit (WM-BUD-10). */
export type LimitDirection = 'increase' | 'lower' | 'keep';

/** Whether wants keep their usual level (WM-BUD-10). */
export interface SavingsCheck {
  averageIncome: number;
  /** What dated goals need saved a month. */
  goalNeed: number;
  /** The higher of goalNeed and 30% of income. */
  required: number;
  /** Limits with wants at their usual level. */
  usualSpend: number;
  onTrack: boolean;
  recommendedSpend: number;
}

export interface SuggestedLimit {
  category: string;
  nature?: CategoryNature;
  currentLimit: number;
  /** Average over the last 3 full months. */
  averageSpend: number;
  /** Average over the full months since January. */
  yearToDateAverage: number;
  trend: SpendTrend;
  pattern: SpendPattern;
  /** The engine's quality-of-life limit from the averages, trend, pattern and nature. */
  recommendedLimit: number;
  suggestedLimit: number;
  /** The suggestion against the current limit. */
  direction: LimitDirection;
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
  savingsCheck: SavingsCheck;
  suggestedLimits: SuggestedLimit[];
  projection: { date: string; balance: number; note?: string }[];
  alerts: AdvisorAlert[];
  goalImpact: { goal: string; impact: string }[];
  forecastedSavings: number;
  unusualVelocity: { category: string; dailyNow: number; dailyAverage: number; ratio: number }[];
  /** What the advisor filed to the Budget page's recommendations (WM-BUD-11). */
  recorded?: RecommendationRecord;
}

export interface CategorySuggestion {
  category: string;
  source: 'rule' | 'history' | 'ai' | 'default';
  /** One of the category's split tags when it fits the payee (WM-CAT-5). */
  splitTag?: string;
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
  type: 'text' | 'tool' | 'form' | 'done' | 'error';
  delta?: string;
  tool?: string;
  status?: 'running' | 'ok' | 'error';
  model?: string;
  message?: string;
  /** A filled form the user can open (WM-FILL-7). */
  form?: FillChatForm;
}
