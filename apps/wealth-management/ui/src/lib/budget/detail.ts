import type { Transaction } from '$lib/api/types';

const key = (category: string) => category.trim().toLowerCase();

/** The day a row counts toward: the rules' effective date when the engine sends one (WM-BUD-4). */
export function budgetDate(tx: Pick<Transaction, 'date' | 'effectiveDate'>): string {
  return tx.effectiveDate ?? tx.date;
}

/** Transactions of one category attributed to a month ("YYYY-MM"), newest first. */
export function categoryTransactions(transactions: Transaction[], category: string, month: string): Transaction[] {
  return transactions.filter((tx) => key(tx.category) === key(category) && budgetDate(tx).startsWith(month));
}

/**
 * Net daily spend (payment - deposit, per WM-RULE-5) for the 30 days ending
 * on `end` ("YYYY-MM-DD"), oldest first; days without spend are zero.
 */
export function dailySpend(
  transactions: Transaction[],
  category: string,
  end: string,
): { date: string; value: number }[] {
  return dailySplitSpend(transactions, category, end, []).map(({ date, rest }) => ({ date, value: rest }));
}

/**
 * Daily spend divided by the category's split tags (WM-CAT-3): `tags[i]` is
 * the day's spend on rows tagged with split tag i, `rest` the rows with none.
 * A row with two split tags counts under the first in the list, so the parts
 * add up to the day. Each part nets refunds and never goes below zero.
 */
export function dailySplitSpend(
  transactions: Transaction[],
  category: string,
  end: string,
  splitTags: string[],
): { date: string; rest: number; tags: number[] }[] {
  const totals = new Map<string, number[]>();
  for (const tx of transactions) {
    if (key(tx.category) !== key(category)) continue;
    const day = tx.date.slice(0, 10);
    const part = splitTagIndex(tx, splitTags);
    const parts = totals.get(day) ?? new Array<number>(splitTags.length + 1).fill(0);
    parts[part] += (tx.payment ?? 0) - (tx.deposit ?? 0);
    totals.set(day, parts);
  }
  const [year, month, day] = end.split('-').map(Number);
  return Array.from({ length: 30 }, (_, index) => {
    const date = new Date(Date.UTC(year, month - 1, day - 29 + index)).toISOString().slice(0, 10);
    const parts = (totals.get(date) ?? new Array<number>(splitTags.length + 1).fill(0)).map((value) =>
      Math.max(0, value),
    );
    return { date, rest: parts[splitTags.length], tags: parts.slice(0, splitTags.length) };
  });
}

/** Index of the first split tag the row carries (case ignored); `splitTags.length` when none. */
function splitTagIndex(tx: Pick<Transaction, 'tags'>, splitTags: string[]): number {
  const tags = (tx.tags ?? []).map((tag) => tag.trim().toLowerCase());
  const index = splitTags.findIndex((tag) => tags.includes(tag));
  return index === -1 ? splitTags.length : index;
}

/** The split tag a row carries, the first in the category's list; undefined when none (WM-CAT-3). */
export function rowSplitTag(tx: Pick<Transaction, 'tags'>, splitTags: string[]): string | undefined {
  return splitTags[splitTagIndex(tx, splitTags)];
}

/** Last day of the month, or today when the month is current. */
export function detailEnd(month: string, today: string): string {
  if (today.startsWith(month)) return today;
  const [year, monthIndex] = month.split('-').map(Number);
  return new Date(Date.UTC(year, monthIndex, 0)).toISOString().slice(0, 10);
}

/**
 * The stacked parts of a category's daily spend: the rest first, then each split tag. Past two tags
 * the others fold into one "Other split tags" part, so the colours stay told apart.
 */
export function splitSeries(
  days: { rest: number; tags: number[] }[],
  splitTags: string[],
): { name: string; part: 'rest' | 'first' | 'second'; values: number[] }[] {
  const series: { name: string; part: 'rest' | 'first' | 'second'; values: number[] }[] = [
    { name: splitTags.length ? 'Rest of the category' : 'Spend', part: 'rest', values: days.map((d) => d.rest) },
  ];
  if (splitTags.length > 0)
    series.push({ name: `#${splitTags[0]}`, part: 'first', values: days.map((d) => d.tags[0]) });
  if (splitTags.length > 1)
    series.push({
      name: splitTags.length === 2 ? `#${splitTags[1]}` : 'Other split tags',
      part: 'second',
      values: days.map((d) => d.tags.slice(1).reduce((sum, value) => sum + value, 0)),
    });
  return series;
}
