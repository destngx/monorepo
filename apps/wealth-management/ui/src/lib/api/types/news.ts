import type { ExchangeRate } from './rates';
import type { MarketPulse } from './market';
import type { DepositRate, PortfolioValuation } from './investments';
import type { RecommendationRecord } from './recommendations';

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

/** What a think tank run analyses (ROADMAP M5): symbols the user chose, else the holdings. */
export interface ThinkTankFocus {
  source: 'holdings' | 'request';
  symbols: string[];
  /** Why a holdings focus is empty, e.g. the valuation failed. */
  note?: string;
}

/** The raw inputs the think tank saw, disclosed as-is (WM-INV-18). */
export interface ThinkTankContext {
  focus?: ThinkTankFocus;
  valuation?: PortfolioValuation;
  valuationError?: string;
  /** Fund performance vs a deposit, aggregated per fund code (WM-INV-23). */
  fundPerformance?: {
    funds: Record<string, unknown>[];
    total?: Record<string, unknown>;
    deposit?: DepositRate;
    note: string;
  };
  fundPerformanceError?: string;
  accounts: unknown[];
  moneyLent: { owedToYou: number; youOwe: number; open: unknown[] };
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
  | {
      type: 'actions';
      phase: InvestmentPhase;
      name: string;
      commands: InvestmentCommand[];
      model?: string;
      /** What the run wrote to the recommendations list (WM-REC-8). */
      recorded?: RecommendationRecord;
    }
  | { type: 'done' }
  | { type: 'error'; message: string };
