// Pure helpers for the market pages (WM-INV-11..16). Every figure is computed
// by the engine; these only parse URL / storage input and format for display.
import {
  TIMEFRAMES,
  type Direction,
  type Momentum,
  type PulseQuote,
  type SeasonalityBucket,
  type SeasonalityType,
  type Timeframe,
  type Trend,
} from '$lib/api/types';

export const DEFAULT_TIMEFRAME: Timeframe = '1d';

export const TIMEFRAME_LABELS: Record<Timeframe, string> = { '1h': '1H', '4h': '4H', '1d': '1D', '1w': '1W' };

/** The engine's pulse instruments, offered as quick picks on the deep-dive. */
export const PULSE_SYMBOLS = [
  'VNINDEX',
  'VN30',
  'VCB',
  'VNM',
  'ACB',
  'TCB',
  'MBB',
  'FPT',
  'HPG',
  'SAB',
  'GAS',
  'USDVND',
  'SJC',
  'VIX',
] as const;

/** A `?tf=` value, falling back to the daily view when missing or unknown. */
export function parseTimeframe(value: string | null | undefined): Timeframe {
  const normalised = (value ?? '').trim().toLowerCase();
  return (TIMEFRAMES as string[]).includes(normalised) ? (normalised as Timeframe) : DEFAULT_TIMEFRAME;
}

/** A `?symbol=` value: uppercase ticker characters only, or '' when unusable. */
export function parseSymbol(value: string | null | undefined): string {
  const normalised = (value ?? '').trim().toUpperCase();
  return /^[A-Z0-9][A-Z0-9._/-]{0,19}$/.test(normalised) ? normalised : '';
}

// Auto-refresh (minutes; 0 = off), persisted per browser.
export const REFRESH_STORAGE_KEY = 'wm.market.autoRefresh';
export const REFRESH_INTERVALS = [0, 1, 5, 15] as const;
export type RefreshInterval = (typeof REFRESH_INTERVALS)[number];

/** A stored interval, or off when the value is missing or not offered. */
export function parseRefreshInterval(value: string | null | undefined): RefreshInterval {
  const minutes = Number(value);
  return (REFRESH_INTERVALS as readonly number[]).includes(minutes) ? (minutes as RefreshInterval) : 0;
}

/** Whether an auto-refresh is owed, e.g. when the tab becomes visible again. */
export function isRefreshDue(lastUpdated: number | null, minutes: number, now: number): boolean {
  if (minutes <= 0) return false;
  return lastUpdated === null || now - lastUpdated >= minutes * 60_000;
}

const priceFormat = new Intl.NumberFormat('en-US', { maximumFractionDigits: 2 });
const smallPriceFormat = new Intl.NumberFormat('en-US', { maximumSignificantDigits: 4 });

/** Instrument prices keep two decimals (indices, FX) and grouping. */
export function formatPrice(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  return Math.abs(value) > 0 && Math.abs(value) < 1 ? smallPriceFormat.format(value) : priceFormat.format(value);
}

export function formatNumber(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '-';
  return value.toFixed(digits);
}

/** Matches the engine's ±0.05% flat band (WM-RULE-9). */
export const FLAT_THRESHOLD_PCT = 0.05;

export function changeDirection(value: number | null | undefined): Direction | null {
  if (value === null || value === undefined || !Number.isFinite(value)) return null;
  if (value > FLAT_THRESHOLD_PCT) return 'up';
  if (value < -FLAT_THRESHOLD_PCT) return 'down';
  return 'flat';
}

const directionText: Record<Direction, string> = {
  up: 'text-emerald-600 dark:text-emerald-400',
  down: 'text-rose-600 dark:text-rose-400',
  flat: 'text-slate-500 dark:text-slate-400',
};

/** Text colour of a percent change. */
export function changeClass(value: number | null | undefined): string {
  const direction = changeDirection(value);
  return direction ? directionText[direction] : directionText.flat;
}

export const momentumLabel: Record<Momentum, string> = {
  fire: 'Strong move',
  stable: 'Steady',
  sleep: 'Quiet',
};

export type Tone = 'neutral' | 'green' | 'orange' | 'amber' | 'red' | 'blue' | 'violet';

export const trendTone: Record<Trend, Tone> = { up: 'green', down: 'red', sideways: 'neutral' };

/** Badge tone of a signal action (LONG, SHORT, EXIT, HOLD/WATCH, ...). */
export function actionTone(action: string): Tone {
  switch (action.toUpperCase()) {
    case 'LONG':
      return 'green';
    case 'SHORT':
    case 'EXIT':
      return 'red';
    case 'TAKE PROFIT':
    case 'REDUCE':
      return 'amber';
    default:
      return 'neutral';
  }
}

export const pairTone: Record<'ALIGNED' | 'CONFLICT' | 'MIXED', Tone> = {
  ALIGNED: 'green',
  CONFLICT: 'red',
  MIXED: 'amber',
};

export const flowTone: Record<'DEFENSIVE' | 'RISK-ON' | 'MIXED', Tone> = {
  DEFENSIVE: 'amber',
  'RISK-ON': 'green',
  MIXED: 'neutral',
};

/** Sentiment 0-100: fear below 35, greed above 65 (WM-RULE-9). */
export function sentimentTone(score: number): Tone {
  if (score > 65) return 'green';
  if (score < 35) return 'red';
  return 'amber';
}

/** RSI zones: overbought above 70, oversold below 30. */
export function rsiTone(rsi: number | null | undefined): Tone {
  if (rsi === null || rsi === undefined) return 'neutral';
  if (rsi >= 70) return 'red';
  if (rsi <= 30) return 'green';
  return 'neutral';
}

/**
 * Heatmap fill for a correlation in [-1, 1]: green for positive, red for
 * negative, stronger with magnitude. n/a (null) cells get no fill.
 */
export function correlationStyle(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '';
  const strength = Math.min(1, Math.abs(value));
  const alpha = Math.round(strength * 0.7 * 100) / 100;
  if (alpha === 0) return '';
  const rgb = value > 0 ? '16 185 129' : '244 63 94';
  return `background-color: rgb(${rgb} / ${alpha})`;
}

export type QuoteSortKey = 'symbol' | 'changePct' | 'dayChangePct' | 'weekChangePct';
export type SortDirection = 'asc' | 'desc';

/**
 * Sorts pulse quotes without mutating them. Failed quotes and missing values
 * always go last so a sort never hides the working rows behind errors.
 */
export function sortQuotes(quotes: PulseQuote[], key: QuoteSortKey | null, direction: SortDirection): PulseQuote[] {
  if (!key) return [...quotes];
  const sign = direction === 'asc' ? 1 : -1;
  return [...quotes].sort((a, b) => {
    if (Boolean(a.error) !== Boolean(b.error)) return a.error ? 1 : -1;
    if (key === 'symbol') return sign * a.symbol.localeCompare(b.symbol);
    const left = a[key];
    const right = b[key];
    if (left === null || right === null) return left === right ? 0 : left === null ? 1 : -1;
    return sign * (left - right);
  });
}

export interface SeasonalityGroup {
  type: SeasonalityType;
  title: string;
  rows: SeasonalityBucket[];
  /** Worst rank in the group; rank 1 is the best. */
  worstRank: number;
}

const seasonalityTitles: Record<SeasonalityType, string> = {
  day: 'Day of week',
  week: 'Week of month',
  month: 'Month of year',
};

/** Buckets grouped per type in calendar order, empty types dropped. */
export function groupSeasonality(buckets: SeasonalityBucket[]): SeasonalityGroup[] {
  return (['day', 'week', 'month'] as const)
    .map((type) => {
      const rows = buckets.filter((bucket) => bucket.type === type).sort((a, b) => a.key - b.key);
      return {
        type,
        title: seasonalityTitles[type],
        rows,
        worstRank: rows.reduce((worst, row) => Math.max(worst, row.rank), 0),
      };
    })
    .filter((group) => group.rows.length > 0);
}

/** 'best' for rank 1, 'worst' for the last rank; only when there is a choice. */
export function rankMark(rank: number, worstRank: number): 'best' | 'worst' | null {
  if (worstRank < 2) return null;
  if (rank === 1) return 'best';
  if (rank === worstRank) return 'worst';
  return null;
}

const timeFormat = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Ho_Chi_Minh',
  hour: '2-digit',
  minute: '2-digit',
  second: '2-digit',
  hour12: false,
});

const dateTimeFormat = new Intl.DateTimeFormat('en-GB', {
  timeZone: 'Asia/Ho_Chi_Minh',
  day: '2-digit',
  month: '2-digit',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  hour12: false,
});

/** Vietnam wall-clock time of a timestamp, e.g. "14:05:09". */
export function formatClock(value: string | null | undefined): string {
  const date = new Date(value ?? '');
  return Number.isNaN(date.getTime()) ? '-' : timeFormat.format(date);
}

/** Vietnam date and time of a timestamp, e.g. "13/10/2026, 14:05". */
export function formatDateTime(value: string | null | undefined): string {
  const date = new Date(value ?? '');
  return Number.isNaN(date.getTime()) ? '-' : dateTimeFormat.format(date);
}

/** The deep-dive link of a symbol at a timeframe (path relative to base). */
export function analysisHref(symbol: string, timeframe: Timeframe): string {
  const params = new URLSearchParams({ symbol, tf: timeframe });
  return `/investments/analysis?${params.toString()}`;
}
