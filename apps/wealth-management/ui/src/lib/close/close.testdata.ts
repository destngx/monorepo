import type { CloseItem, MonthlyClose } from '$lib/api/types';

// Invented figures for the monthly close tests.

const item = (
  key: CloseItem['key'],
  title: string,
  state: CloseItem['state'],
  extra: Partial<CloseItem> = {},
): CloseItem => ({
  key,
  title,
  state,
  detail: `${title} detail.`,
  count: null,
  amount: null,
  target: null,
  ...extra,
});

export function monthlyClose(overrides: Partial<MonthlyClose> = {}): MonthlyClose {
  return {
    month: '2026-10',
    asOf: '2026-10-31',
    monthEnded: true,
    canWriteSnapshot: true,
    items: [
      item('drafts', 'Approve capture drafts', 'action', { count: 2 }),
      item('categorize', 'Categorize transactions', 'done', { count: 0, amount: 0 }),
      item('budget', 'Budget vs actual', 'action', { count: 1, amount: 3_000_000, target: 20_000_000 }),
      item('goals', 'Surplus to goals', 'unavailable'),
      item('moneyLent', 'Chase money lent', 'done', { count: 0, amount: 0 }),
      item('recommendations', 'Review recommendations', 'action', { count: 2, amount: 5_000_000 }),
      item('snapshot', 'Write the snapshot', 'action'),
    ],
    done: 2,
    checked: 7,
    snapshot: null,
    ...overrides,
  };
}
