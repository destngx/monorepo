// AI form filling (§10.5): the engine reads a sentence or images into a
// form's fields; these helpers merge its answer into the form state. The
// user still saves through the form's normal path (D46).

import type { FillFormKey, FillNote, FillResult, FillValues } from '$lib/api/types';

/** Forms that take images as well as text (WM-FILL-15). */
const TEXT_ONLY: ReadonlySet<FillFormKey> = new Set([
  'budget-move',
  'budget-limits',
  'category-definition',
  'allocation',
]);

export function takesImages(form: FillFormKey): boolean {
  return !TEXT_ONLY.has(form);
}

export const FILL_MAX_TEXT = 2000;
export const FILL_MAX_IMAGES = 4;

/** An example sentence per form, shown as the box's placeholder. */
export const FILL_EXAMPLES: Record<FillFormKey, string> = {
  transaction: 'e.g. coffee 45k with MoMo yesterday',
  goal: 'e.g. save 50M for a trip to Japan by June 2027, after the emergency fund',
  recurring: 'e.g. Netflix 260k on the 15th every month, paid by card',
  'budget-limits': 'e.g. food 6M, transport 2M, cut shopping to 3M',
  'budget-move': 'e.g. move 1M from shopping to food',
  'category-definition': 'e.g. groceries and eating out, not alcohol; split by work and family',
  allocation: 'e.g. 60% funds, 30% cash, 10% crypto',
  account: 'e.g. new savings account at Bank B, 20M, VND',
  'minimum-balance': 'e.g. keep 2M in Bank A to avoid fees',
  'account-savings': 'e.g. my wallet savings is emergency money',
  'best-card': 'e.g. paying 1.2M at the supermarket',
  stablecoin: 'e.g. 10k USDT at 5%, switch to 9%, 0.1% fee each way, 90 days',
  'fund-trade': 'e.g. bought 120.5 units of VESAF today',
  'crypto-holdings': 'e.g. 0.012 BTC and 1,250 USDT on Binance',
  card: 'e.g. statement on the 20th, pay by the 5th, limit 50M',
  'cashback-program': 'e.g. 5% at supermarkets (MCC 5411) capped at 300k, 0.5% on the rest',
  merchant: 'e.g. the supermarket is MCC 5411, from the receipt',
};

/**
 * Merges the answer's values into a form: only keys the form has, each
 * through `convert` when given (e.g. numbers into a text field).
 */
export function applyFill<T extends object>(
  form: T,
  values: FillValues,
  convert: Partial<Record<keyof T & string, (value: unknown) => unknown>> = {},
): T {
  const next = { ...form } as Record<string, unknown>;
  for (const [key, value] of Object.entries(values)) {
    if (!(key in next)) continue;
    const converter = (convert as Record<string, ((value: unknown) => unknown) | undefined>)[key];
    next[key] = converter ? converter(value) : value;
  }
  return next as T;
}

/** The values of the fields the answer set, keyed by field, for FillMarks. */
export function filledValues(result: FillResult): FillValues {
  const marked: FillValues = {};
  for (const field of result.filled) {
    if (field in result.values) marked[field] = result.values[field];
  }
  return marked;
}

/** Notes about one field, or the general notes when field is undefined. */
export function notesFor(notes: FillNote[], field?: string, row?: number): FillNote[] {
  return notes.filter((note) => note.field === field && note.row === row);
}

/** Compares a filled value with what the form holds now. */
export function sameValue(a: unknown, b: unknown): boolean {
  return normalise(a) === normalise(b);
}

function normalise(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (Array.isArray(value)) return value.map(normalise).join(',');
  if (typeof value === 'number') return Number.isFinite(value) ? String(value) : '';
  if (typeof value === 'string') return value.trim();
  return JSON.stringify(value);
}

/** A number for a nullable number field; anything else is null. */
export function asNumber(value: unknown): number | null {
  return typeof value === 'number' && Number.isFinite(value) ? value : null;
}

/** Text for a text field, e.g. a number written into an amount input. */
export function asText(value: unknown): string {
  if (value === null || value === undefined) return '';
  if (Array.isArray(value)) return value.map(asText).join(', ');
  if (typeof value === 'string') return value;
  if (typeof value === 'number' || typeof value === 'boolean') return String(value);
  return JSON.stringify(value);
}

/** What a fill set, e.g. "Filled 3 fields and 2 rules"; empty when nothing. */
export function fillCount(result: FillResult, rowsLabel = 'row'): string {
  const rows = result.rows?.length ?? 0;
  const fields = result.filled.filter((field) => field !== 'rows').length;
  const parts: string[] = [];
  if (fields) parts.push(`${fields} ${fields === 1 ? 'field' : 'fields'}`);
  if (rows) parts.push(`${rows} ${rows === 1 ? rowsLabel : `${rowsLabel}s`}`);
  return parts.length ? `Filled ${parts.join(' and ')}` : '';
}

/** What the summary line says after a fill. */
export function fillSummary(result: FillResult, rowsLabel = 'row'): string {
  const count = fillCount(result, rowsLabel);
  return count ? `${count}. Check them, then save.` : 'Nothing to fill from that.';
}
