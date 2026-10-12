import { describe, expect, it } from 'vitest';
import { wallRows } from './history';

describe('wallRows', () => {
  const history = {
    months: ['2026-08', '2026-09', '2026-10'],
    rows: [
      {
        category: 'Fun',
        values: [
          { month: '2026-08', limit: 100, spent: 50 },
          { month: '2026-10', limit: 100, spent: 75 },
        ],
      },
      {
        category: 'Food',
        values: [
          { month: '2026-08', limit: 100, spent: 120 },
          { month: '2026-09', limit: 0, spent: 30 },
          { month: '2026-10', limit: 100, spent: 89 },
        ],
      },
      { category: 'Unused', values: [{ month: '2026-09', limit: 0, spent: 0 }] },
    ],
  };

  it('fills every month, rates each cell and drops empty categories', () => {
    const rows = wallRows(history);
    expect(rows.map((row) => row.category)).toEqual(['Food', 'Fun']);
    expect(rows[0].cells.map((cell) => cell.status)).toEqual(['urgent', 'unbudgeted', 'warning']);
    expect(rows[1].cells.map((cell) => cell.status)).toEqual(['healthy', 'empty', 'warning']);
    expect(rows[1].cells[1]).toMatchObject({ month: '2026-09', limit: 0, spent: 0, percent: null });
    expect(rows[0].totalSpent).toBe(239);
  });
});
