import type { NewsArticle, NewsBias, NewsReport, NewsSentiment } from '$lib/api/types';

// News Scanner helpers (WM-INV-17). The engine classifies articles and
// derives the bias; the UI only stores, orders and displays reports.

/** Scanned reports last for the browser tab so the Think Tank can include them. */
export const NEWS_STORAGE_KEY = 'wm.news.reports';

type Tone = 'green' | 'red' | 'neutral';

export const biasTone: Record<NewsBias, Tone> = { BULLISH: 'green', BEARISH: 'red', NEUTRAL: 'neutral' };
export const sentimentTone: Record<NewsSentiment, Tone> = { Positive: 'green', Negative: 'red', Neutral: 'neutral' };

/** Collapses whitespace like the engine does, so "a  b" and "a b" are one topic. */
export function normalizeTopic(topic: string): string {
  return topic.trim().split(/\s+/).filter(Boolean).join(' ');
}

/** Case-insensitive key of a topic. */
export function topicKey(topic: string): string {
  return normalizeTopic(topic).toLowerCase();
}

/** Replaces the report of the same topic in place, or appends a new one. */
export function upsertReport(reports: NewsReport[], report: NewsReport): NewsReport[] {
  const key = topicKey(report.topic);
  const index = reports.findIndex((r) => topicKey(r.topic) === key);
  if (index < 0) return [...reports, report];
  return reports.map((r, i) => (i === index ? report : r));
}

export function removeReport(reports: NewsReport[], topic: string): NewsReport[] {
  const key = topicKey(topic);
  return reports.filter((r) => topicKey(r.topic) !== key);
}

function isReport(value: unknown): value is NewsReport {
  if (!value || typeof value !== 'object') return false;
  const r = value as Partial<NewsReport>;
  return (
    typeof r.topic === 'string' &&
    typeof r.bias === 'string' &&
    Array.isArray(r.articles) &&
    !!r.counts &&
    typeof r.counts === 'object' &&
    typeof r.generatedAt === 'string'
  );
}

/** Reads stored reports, dropping anything malformed instead of failing the page. */
export function parseStoredReports(raw: string | null): NewsReport[] {
  if (!raw) return [];
  try {
    const value: unknown = JSON.parse(raw);
    return Array.isArray(value) ? value.filter(isReport) : [];
  } catch {
    return [];
  }
}

/** The article's host without "www.", falling back to the source the engine gave. */
export function sourceHost(article: Pick<NewsArticle, 'url' | 'source'>): string {
  try {
    return new URL(article.url).hostname.replace(/^www\./, '') || article.source;
  } catch {
    return article.source || article.url;
  }
}

/** Positive / negative / neutral shares in percent; all zero for an empty report. */
export function countShares(counts: NewsReport['counts']): { positive: number; negative: number; neutral: number } {
  const total = counts.positive + counts.negative + counts.neutral;
  if (total <= 0) return { positive: 0, negative: 0, neutral: 0 };
  return {
    positive: (counts.positive / total) * 100,
    negative: (counts.negative / total) * 100,
    neutral: (counts.neutral / total) * 100,
  };
}

/** Clamps a 0-100 model score for bar widths. */
export function clampScore(value: number): number {
  return Number.isFinite(value) ? Math.min(100, Math.max(0, value)) : 0;
}

const pad = (n: number) => String(n).padStart(2, '0');

/** A timestamp as local dd/MM/yyyy HH:mm; unparseable input is shown as-is. */
export function formatTimestamp(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return `${pad(date.getDate())}/${pad(date.getMonth() + 1)}/${date.getFullYear()} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}
