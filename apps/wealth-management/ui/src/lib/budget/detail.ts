import type { Transaction } from '$lib/api/types';

const key = (category: string) => category.trim().toLowerCase();

/** Transactions of one category within a month ("YYYY-MM"), newest first. */
export function categoryTransactions(transactions: Transaction[], category: string, month: string): Transaction[] {
  return transactions.filter((tx) => key(tx.category) === key(category) && tx.date.startsWith(month));
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
  const totals = new Map<string, number>();
  for (const tx of transactions) {
    if (key(tx.category) !== key(category)) continue;
    const day = tx.date.slice(0, 10);
    totals.set(day, (totals.get(day) ?? 0) + (tx.payment ?? 0) - (tx.deposit ?? 0));
  }
  const [year, month, day] = end.split('-').map(Number);
  return Array.from({ length: 30 }, (_, index) => {
    const date = new Date(Date.UTC(year, month - 1, day - 29 + index)).toISOString().slice(0, 10);
    return { date, value: Math.max(0, totals.get(date) ?? 0) };
  });
}

/** Last day of the month, or today when the month is current. */
export function detailEnd(month: string, today: string): string {
  if (today.startsWith(month)) return today;
  const [year, monthIndex] = month.split('-').map(Number);
  return new Date(Date.UTC(year, monthIndex, 0)).toISOString().slice(0, 10);
}
