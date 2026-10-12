export type Json = Record<string, unknown>;

export const isObject = (value: unknown): value is Json =>
  typeof value === 'object' && value !== null && !Array.isArray(value);
export const objectAt = (value: unknown): Json => (isObject(value) ? value : {});
export const arrayAt = (value: unknown): unknown[] => (Array.isArray(value) ? value : []);

export function num(value: unknown): number | null {
  if (typeof value === 'number') return Number.isFinite(value) ? value : null;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

export function str(value: unknown): string | null {
  if (typeof value === 'string') return value.trim() || null;
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return null;
}

/** An absolute http(s) URL, or null; keeps odd values out of href and src. */
export function url(value: unknown): string | null {
  const text = str(value);
  if (!text) return null;
  try {
    const parsed = new URL(text);
    return parsed.protocol === 'https:' || parsed.protocol === 'http:' ? parsed.href : null;
  } catch {
    return null;
  }
}

/** fmarket wraps payloads as `{status, code, message, data, extra}`. */
export const dataOf = (payload: unknown): unknown => objectAt(payload).data;

/** Epoch milliseconds as a calendar date in Vietnam (UTC+7, no DST). */
export function vnDate(ms: number): string {
  return new Date(ms + 7 * 3_600_000).toISOString().slice(0, 10);
}

export function dateOf(value: unknown): string | null {
  const ms = num(value);
  if (ms !== null) return vnDate(ms);
  const text = str(value);
  return text && /^\d{4}-\d{2}-\d{2}/.test(text) ? text.slice(0, 10) : null;
}
