import type { BudgetHistory, BudgetStatus } from '$lib/api/types';

// "The Wall" (WM-BUD-7): categories by month, each cell spent against limit.

export interface WallCell {
  month: string;
  limit: number;
  spent: number;
  /** Spent as a percent of the limit; null when there is no limit. */
  percent: number | null;
  status: BudgetStatus | 'unbudgeted' | 'empty';
}

export interface WallRow {
  category: string;
  cells: WallCell[];
  totalSpent: number;
}

// The engine's status bands (WM-BUD-3, domain.BudgetStatusFor): warning from 70%, urgent from 90%.
function cellStatus(limit: number, spent: number): WallCell['status'] {
  if (limit <= 0) return spent > 0 ? 'unbudgeted' : 'empty';
  const percent = (spent / limit) * 100;
  if (percent >= 90) return 'urgent';
  if (percent >= 70) return 'warning';
  return 'healthy';
}

/** One row per category with a cell for every month (missing months are empty), biggest spenders first. */
export function wallRows(history: BudgetHistory): WallRow[] {
  return history.rows
    .map((row) => {
      const byMonth = new Map(row.values.map((value) => [value.month, value]));
      const cells = history.months.map((month): WallCell => {
        const value = byMonth.get(month);
        const limit = value?.limit ?? 0;
        const spent = value?.spent ?? 0;
        return {
          month,
          limit,
          spent,
          percent: limit > 0 ? (spent / limit) * 100 : null,
          status: cellStatus(limit, spent),
        };
      });
      return { category: row.category, cells, totalSpent: cells.reduce((sum, cell) => sum + cell.spent, 0) };
    })
    .filter((row) => row.cells.some((cell) => cell.limit > 0 || cell.spent > 0))
    .sort((a, b) => b.totalSpent - a.totalSpent || a.category.localeCompare(b.category));
}
