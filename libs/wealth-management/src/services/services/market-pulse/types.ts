export interface MarketAsset {
  symbol: string;
  name: string;
  market: 'US' | 'VN';
  price: number;
  percentChange: number;
  dayChange: number;
  weekChange: number;
  direction: 'up' | 'down' | 'flat';
  momentum: 'fire' | 'stable' | 'sleep';
  closes?: number[]; // Added historical closes for real calculations
  highs?: number[];
  lows?: number[];
  timestamps?: number[]; // Added historical timestamps for seasonality
}

export interface Technicals {
  cycle: {
    phase: 'Accumulation' | 'Markup' | 'Distribution' | 'Mark-Down' | 'Decline';
    description: string;
    descriptionVi: string;
    strength: number; // 0-100
    confidence: number;
    phases?: { label: string; value: number; color?: string }[]; // For Donut chart
  };
  trend: {
    direction: string;
    directionEn: string;
    strength: number;
    confidence: number;
  };
  ict: {
    fvgs: { type: string; top: number; bottom: number; gap: number }[];
    orderBlocks: any[];
  };
  sentiment: {
    score: number;
    label: string;
    labelVi: string;
  };
  indicators: {
    rsi?: number;
    ema20?: number;
    ema50?: number;
    ema200?: number;
    sma20?: number;
    sma50?: number;
    sma200?: number;
  };
  signals?: {
    action: 'SHORT' | 'LONG' | 'AVOID' | 'EXIT' | 'REDUCE' | 'TAKE PROFIT' | 'HOLD/WATCH';
    actionVi: string;
    entry: number;
    stopLoss: number;
    takeProfit: number;
    rr: number; // Risk:Reward ratio
    confidence: number;
    reasons: string[];
    reasonsVi: string[];
    optimalEntry?: {
      price: number;
      pullbackPercent: number;
      pullbackAmount: number;
    };
  };
  timeframeRelationships?: {
    pair: string; // e.g., "1wk -> 1d"
    status: 'STRONG' | 'WEAK' | 'CHOPIED' | 'ALIGNED';
    relationship: string;
    relationshipVi: string;
    advice: string;
    adviceVi: string;
  }[];
  entryTimingScore?: {
    overall: number; // 0-10
    higherTfSupport: number; // 0-6
    lowerTfConfirm: number; // 0-4
  };
  supportResistance: {
    symbol: string;
    support: number[];
    resistance: number[];
    bollingerUpper: number;
    bollingerLower: number;
    bollingerMid: number;
  }[];
  seasonality: {
    rank: number;
    name: string;
    label: string;
    return: number;
    winRate: number;
    pf: number; // Profit Factor
    stdDev: number;
    score: number;
    n: number;
    type: 'day' | 'week' | 'month';
  }[];
  n: number;
  dateRange: string;
}

export interface Valuation {
  dcf: {
    symbol: string;
    fairValue: number;
    upside: number;
    assumptions: { label: string; value: string }[];
  }[];
  monteCarlo: {
    symbol: string;
    p10: number;
    p50: number;
    p90: number;
    iterations: number;
  }[];
  sectorComparison: {
    symbol: string;
    sector: string;
    metrics: {
      label: string;
      asset: number;
      avg: number;
    }[];
  }[];
}

export interface MarketState {
  assets: MarketAsset[];
  drivers: {
    topMovers: { symbol: string; change: number }[];
    summaryEn: string;
    summaryVi: string;
    capitalFlowSignal?: 'DEFENSIVE' | 'RISK-ON' | 'MIXED';
    correlationSignalEn?: string;
    correlationSignalVi?: string;
  };
  capitalFlow: {
    signal: 'DEFENSIVE' | 'RISK-ON' | 'MIXED';
    summaryEn: string;
    summaryVi: string;
  };
  scenarios?: {
    name: string;
    regime: 'Risk-ON' | 'Risk-OFF' | 'Crisis' | 'Stagflation' | 'Goldilocks';
    confidence: number;
    summaryEn: string;
    summaryVi: string;
    actionEn?: string;
    actionVi?: string;
  }[];
  correlationMatrix: number[][];
  assetList: string[]; // For matrix headers
  technicals?: Technicals;
  valuation?: Valuation;
}

export interface MarketPulseResponse {
  us: MarketState;
  vn: MarketState;
  lastUpdated: string;
}
