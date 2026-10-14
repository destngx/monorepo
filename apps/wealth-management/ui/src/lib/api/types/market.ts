// Market intelligence (WM-INV-11..18). Every figure below is computed by the
// engine from candles; nothing here comes from a model.

import type { DataSource } from './source';

export type Timeframe = '1h' | '4h' | '1d' | '1w';
export const TIMEFRAMES: Timeframe[] = ['1h', '4h', '1d', '1w'];

export type Trend = 'up' | 'down' | 'sideways';
export type Direction = 'up' | 'down' | 'flat';
export type Momentum = 'fire' | 'stable' | 'sleep';

export interface PulseQuote {
  symbol: string;
  name: string;
  provider?: string;
  source?: DataSource;
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
  source?: DataSource;
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
  source?: DataSource;
  currency?: string;
  technicals: Technicals;
  confluence: Confluence;
  seasonality: SeasonalityBucket[];
  risk: RiskStats | null;
  notes: string[];
  generatedAt: string;
}
