import { describe, expect, it } from 'vitest';
import type { PulseQuote, SeasonalityBucket } from '$lib/api/types';
import {
  actionTone,
  analysisHref,
  changeClass,
  changeDirection,
  correlationStyle,
  formatClock,
  formatPrice,
  groupSeasonality,
  isRefreshDue,
  parseRefreshInterval,
  parseSymbol,
  parseTimeframe,
  rankMark,
  sentimentTone,
  sortQuotes,
} from './market';

describe('parseTimeframe', () => {
  it('accepts known timeframes in any case', () => {
    expect(parseTimeframe('4h')).toBe('4h');
    expect(parseTimeframe(' 1W ')).toBe('1w');
  });

  it('falls back to daily for missing or unknown values', () => {
    expect(parseTimeframe(null)).toBe('1d');
    expect(parseTimeframe('')).toBe('1d');
    expect(parseTimeframe('15m')).toBe('1d');
  });
});

describe('parseSymbol', () => {
  it('uppercases and trims tickers', () => {
    expect(parseSymbol(' fpt ')).toBe('FPT');
    expect(parseSymbol('usdvnd')).toBe('USDVND');
  });

  it('rejects empty or unsafe input', () => {
    expect(parseSymbol(null)).toBe('');
    expect(parseSymbol('FPT; DROP')).toBe('');
    expect(parseSymbol('-FPT')).toBe('');
  });
});

describe('auto-refresh', () => {
  it('parses only the offered intervals', () => {
    expect(parseRefreshInterval('5')).toBe(5);
    expect(parseRefreshInterval('15')).toBe(15);
    expect(parseRefreshInterval('7')).toBe(0);
    expect(parseRefreshInterval('abc')).toBe(0);
    expect(parseRefreshInterval(null)).toBe(0);
  });

  it('is due once the interval has elapsed', () => {
    expect(isRefreshDue(0, 1, 59_999)).toBe(false);
    expect(isRefreshDue(0, 1, 60_000)).toBe(true);
    expect(isRefreshDue(null, 5, 0)).toBe(true);
  });

  it('is never due when off', () => {
    expect(isRefreshDue(null, 0, 1e12)).toBe(false);
  });
});

describe('formatting', () => {
  it('formats prices with grouping and two decimals', () => {
    expect(formatPrice(1280.456)).toBe('1,280.46');
    expect(formatPrice(25400)).toBe('25,400');
    expect(formatPrice(0.012345)).toBe('0.01235');
    expect(formatPrice(null)).toBe('-');
  });

  it('uses the engine flat band for change direction', () => {
    expect(changeDirection(0.05)).toBe('flat');
    expect(changeDirection(0.06)).toBe('up');
    expect(changeDirection(-0.2)).toBe('down');
    expect(changeDirection(null)).toBeNull();
    expect(changeClass(1)).toContain('emerald');
    expect(changeClass(-1)).toContain('rose');
    expect(changeClass(null)).toContain('stone');
  });

  it('shows Vietnam wall-clock time', () => {
    expect(formatClock('2026-10-13T07:05:09Z')).toBe('14:05:09');
    expect(formatClock('nope')).toBe('-');
  });

  it('builds deep-dive links', () => {
    expect(analysisHref('FPT', '4h')).toBe('/investments/analysis?symbol=FPT&tf=4h');
  });
});

describe('tones', () => {
  it('maps signal actions', () => {
    expect(actionTone('LONG')).toBe('green');
    expect(actionTone('EXIT')).toBe('red');
    expect(actionTone('TAKE PROFIT')).toBe('amber');
    expect(actionTone('HOLD/WATCH')).toBe('neutral');
  });

  it('maps sentiment bands', () => {
    expect(sentimentTone(20)).toBe('red');
    expect(sentimentTone(50)).toBe('amber');
    expect(sentimentTone(80)).toBe('green');
  });
});

describe('correlationStyle', () => {
  it('colours positive green and negative red by magnitude', () => {
    expect(correlationStyle(1)).toBe('background-color: rgb(16 185 129 / 0.7)');
    expect(correlationStyle(-0.5)).toBe('background-color: rgb(244 63 94 / 0.35)');
  });

  it('leaves n/a and zero cells neutral', () => {
    expect(correlationStyle(null)).toBe('');
    expect(correlationStyle(0)).toBe('');
    expect(correlationStyle(Number.NaN)).toBe('');
  });
});

const quote = (symbol: string, changePct: number, extra: Partial<PulseQuote> = {}): PulseQuote => ({
  symbol,
  name: symbol,
  price: 1,
  changePct,
  dayChangePct: changePct,
  weekChangePct: null,
  direction: 'flat',
  momentum: 'stable',
  candles: 10,
  ...extra,
});

describe('sortQuotes', () => {
  const quotes = [quote('B', 1), quote('A', -2), quote('ERR', 0, { error: 'down' }), quote('C', 3)];

  it('keeps the engine order without a key', () => {
    expect(sortQuotes(quotes, null, 'desc').map((q) => q.symbol)).toEqual(['B', 'A', 'ERR', 'C']);
  });

  it('sorts by change with failed quotes last', () => {
    expect(sortQuotes(quotes, 'changePct', 'desc').map((q) => q.symbol)).toEqual(['C', 'B', 'A', 'ERR']);
    expect(sortQuotes(quotes, 'changePct', 'asc').map((q) => q.symbol)).toEqual(['A', 'B', 'C', 'ERR']);
  });

  it('sorts by symbol', () => {
    expect(sortQuotes(quotes, 'symbol', 'asc').map((q) => q.symbol)).toEqual(['A', 'B', 'C', 'ERR']);
  });

  it('puts missing values last', () => {
    const rows = [quote('X', 1, { dayChangePct: null }), quote('Y', 1, { dayChangePct: -5 })];
    expect(sortQuotes(rows, 'dayChangePct', 'desc').map((q) => q.symbol)).toEqual(['Y', 'X']);
  });

  it('does not mutate the input', () => {
    sortQuotes(quotes, 'symbol', 'asc');
    expect(quotes[0].symbol).toBe('B');
  });
});

describe('seasonality', () => {
  const bucket = (type: SeasonalityBucket['type'], key: number, rank: number): SeasonalityBucket => ({
    type,
    key,
    label: `${type}${key}`,
    rank,
    avgReturn: 0,
    winRate: 50,
    profitFactor: 1,
    stdDev: 1,
    score: 0,
    n: 5,
  });

  it('groups by type in calendar order and drops empty types', () => {
    const groups = groupSeasonality([bucket('month', 2, 1), bucket('day', 3, 2), bucket('day', 1, 1)]);
    expect(groups.map((g) => g.type)).toEqual(['day', 'month']);
    expect(groups[0].rows.map((r) => r.key)).toEqual([1, 3]);
    expect(groups[0].worstRank).toBe(2);
  });

  it('marks best and worst only when there is a choice', () => {
    expect(rankMark(1, 5)).toBe('best');
    expect(rankMark(5, 5)).toBe('worst');
    expect(rankMark(3, 5)).toBeNull();
    expect(rankMark(1, 1)).toBeNull();
  });
});
