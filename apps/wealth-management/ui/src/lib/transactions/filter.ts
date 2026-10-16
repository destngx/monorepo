import type { Transaction } from '$lib/api/types';

export type SortKey = 'date' | 'accountName' | 'payee' | 'category' | 'payment' | 'deposit';
export type SortDir = 'asc' | 'desc';

/** Case-insensitive search over payee, account, category, memo and tags (WM-TX-2). */
export function searchTransactions(transactions: Transaction[], query: string): Transaction[] {
  const needle = query.trim().toLowerCase();
  if (!needle) return transactions;
  return transactions.filter((tx) =>
    [tx.payee, tx.accountName, tx.category, tx.memo ?? '', ...tx.tags].some((field) =>
      field.toLowerCase().includes(needle),
    ),
  );
}

/**
 * Rows that still need a category, as the engine counts them (domain
 * NeedsCategory): blank, "Uncategorized", or a category the Categories tab
 * does not know (no type).
 */
export function needsCategory(tx: Pick<Transaction, 'category' | 'categoryType'>): boolean {
  const category = tx.category.trim().toLowerCase();
  return category === '' || category === 'uncategorized' || !tx.categoryType;
}

export function sortTransactions(transactions: Transaction[], key: SortKey, dir: SortDir): Transaction[] {
  const sign = dir === 'asc' ? 1 : -1;
  const value = (tx: Transaction): string | number => {
    if (key === 'payment' || key === 'deposit') return tx[key] ?? 0;
    return (tx[key] ?? '').toString().toLowerCase();
  };
  // Stable sort keeps the engine's newest-first order among equal keys.
  return transactions
    .map((tx, index) => ({ tx, index }))
    .sort((a, b) => {
      const va = value(a.tx);
      const vb = value(b.tx);
      if (va < vb) return -sign;
      if (va > vb) return sign;
      return a.index - b.index;
    })
    .map(({ tx }) => tx);
}

export function paginate<T>(items: T[], page: number, size: number): { items: T[]; pages: number; page: number } {
  const pages = Math.max(1, Math.ceil(items.length / size));
  const current = Math.min(Math.max(page, 1), pages);
  return { items: items.slice((current - 1) * size, current * size), pages, page: current };
}
