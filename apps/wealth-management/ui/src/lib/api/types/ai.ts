import type { CaptureLimits } from './capture';

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
}

export type ReviewKind = 'account-review' | 'budget-review' | 'transaction-review' | 'credit-summary' | 'year-review';

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
