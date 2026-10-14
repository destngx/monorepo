import type { DataSource } from './api/types';
import { formatDateTime } from './format';

/**
 * How a market figure was served (WM-INV-15): `live` when the first provider
 * in the chain answered, `fallback` when a later one did, `stale` when every
 * provider failed and the market data server served its last-good copy.
 */
export type SourceState = 'live' | 'fallback' | 'stale';

export function sourceState(source: DataSource): SourceState {
  if (source.stale) return 'stale';
  return source.fallback?.length ? 'fallback' : 'live';
}

/** The badge text: provider, an optional detail such as the rate basis, and the age of a stale copy. */
export function sourceLabel(source: DataSource, detail?: string): string {
  const parts = [source.provider, detail];
  if (source.stale) parts.push(`as of ${formatDateTime(source.asOf)}`);
  return parts.filter(Boolean).join(' · ');
}

/** The badge tooltip: who answered, what failed first and when the data was read. */
export function sourceTitle(source: DataSource, detail?: string): string {
  const served = detail ? `${source.provider} (${detail})` : source.provider;
  const read = formatDateTime(source.asOf);
  const lines = source.stale
    ? [`Every provider failed. Showing the last good copy from ${served}, read ${read}.`]
    : [`Served by ${served}${source.asOf ? `, read ${read}` : ''}.`];
  if (source.fallback?.length) lines.push(`Failed first: ${source.fallback.join('; ')}`);
  return lines.join('\n');
}

/** The `source` of an fmarket action payload (`{data, extra, source}`), if it has a usable one. */
export function sourceOf(payload: unknown): DataSource | undefined {
  if (typeof payload !== 'object' || payload === null) return undefined;
  const raw = (payload as { source?: unknown }).source;
  if (typeof raw !== 'object' || raw === null) return undefined;
  const { provider, fallback, stale, asOf } = raw as Record<string, unknown>;
  if (typeof provider !== 'string' || !provider) return undefined;
  const source: DataSource = { provider };
  const failed = Array.isArray(fallback) ? fallback.filter((item): item is string => typeof item === 'string') : [];
  if (failed.length) source.fallback = failed;
  if (stale === true) source.stale = true;
  if (typeof asOf === 'string' && asOf) source.asOf = asOf;
  return source;
}
