import type { ExchangeRate } from './rates';
import type { MarketPulse } from './market';
import type { PortfolioValuation } from './investments';

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
  | { type: 'actions'; phase: InvestmentPhase; name: string; commands: InvestmentCommand[]; model?: string }
  | { type: 'done' }
  | { type: 'error'; message: string };
