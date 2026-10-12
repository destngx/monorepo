import { describe, expect, it } from 'vitest';
import type { NewsReport } from '$lib/api/types';
import {
  clampScore,
  countShares,
  formatTimestamp,
  normalizeTopic,
  parseStoredReports,
  removeReport,
  sourceHost,
  upsertReport,
} from './news';

const report = (topic: string, bias: NewsReport['bias'] = 'NEUTRAL'): NewsReport => ({
  topic,
  query: `${topic} news`,
  bias,
  counts: { positive: 0, negative: 0, neutral: 0 },
  articles: [],
  provider: 'tavily',
  generatedAt: '2026-10-13T08:30:00Z',
});

describe('news reports store helpers', () => {
  it('normalises topic whitespace', () => {
    expect(normalizeTopic('  Gold   prices \n today ')).toBe('Gold prices today');
  });

  it('replaces the report of the same topic in place, case-insensitively', () => {
    const reports = [report('Gold'), report('Rates')];
    const next = upsertReport(reports, report('  gold ', 'BULLISH'));
    expect(next.map((r) => r.topic)).toEqual(['  gold ', 'Rates']);
    expect(next[0].bias).toBe('BULLISH');
    expect(reports[0].bias).toBe('NEUTRAL');
  });

  it('appends a new topic and removes one', () => {
    const next = upsertReport([report('Gold')], report('Rates'));
    expect(next.map((r) => r.topic)).toEqual(['Gold', 'Rates']);
    expect(removeReport(next, 'GOLD').map((r) => r.topic)).toEqual(['Rates']);
  });

  it('parses stored reports and drops malformed entries', () => {
    const raw = JSON.stringify([report('Gold'), { topic: 'broken' }, 42]);
    expect(parseStoredReports(raw).map((r) => r.topic)).toEqual(['Gold']);
    expect(parseStoredReports('not json')).toEqual([]);
    expect(parseStoredReports('{}')).toEqual([]);
    expect(parseStoredReports(null)).toEqual([]);
  });
});

describe('news display helpers', () => {
  it('shows the host without www and falls back to the source', () => {
    expect(sourceHost({ url: 'https://www.reuters.com/markets/x', source: 'Reuters' })).toBe('reuters.com');
    expect(sourceHost({ url: 'not a url', source: 'VnExpress' })).toBe('VnExpress');
  });

  it('computes count shares and handles an empty report', () => {
    expect(countShares({ positive: 2, negative: 1, neutral: 1 })).toEqual({ positive: 50, negative: 25, neutral: 25 });
    expect(countShares({ positive: 0, negative: 0, neutral: 0 })).toEqual({ positive: 0, negative: 0, neutral: 0 });
  });

  it('clamps scores to 0-100', () => {
    expect(clampScore(140)).toBe(100);
    expect(clampScore(-3)).toBe(0);
    expect(clampScore(Number.NaN)).toBe(0);
    expect(clampScore(42)).toBe(42);
  });

  it('formats timestamps locally and keeps unparseable input', () => {
    const local = new Date(2026, 9, 13, 9, 5);
    expect(formatTimestamp(local.toISOString())).toBe('13/10/2026 09:05');
    expect(formatTimestamp('yesterday')).toBe('yesterday');
  });
});
