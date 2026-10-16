import type { NeedsYou, NeedsYouItem } from '$lib/api/types';

/** An invented inbox item for tests. */
export const needsYouItem = (overrides: Partial<NeedsYouItem> = {}): NeedsYouItem => ({
  id: 'card-due',
  kind: 'card-due',
  status: 'serious',
  headline: '2 card payments due, the first tomorrow',
  detail: 'Pay the statement balance in full by the due date to avoid interest.',
  amountVnd: 8_000_000,
  impactVnd: null,
  dueDate: '2026-10-18',
  count: 2,
  route: '/bills',
  ...overrides,
});

/** An invented inbox, ranked as the engine ranks it. */
export const needsYou = (overrides: Partial<NeedsYou> = {}): NeedsYou => ({
  asOf: '2026-10-17',
  items: [
    needsYouItem({
      id: 'bill-overdue',
      kind: 'bill-overdue',
      status: 'critical',
      headline: '1 bill is overdue',
      amountVnd: 300_000,
      dueDate: '2026-10-05',
      count: 1,
    }),
    needsYouItem(),
    needsYouItem({
      id: 'goal-plan-short',
      kind: 'goal-plan-short',
      status: 'warning',
      headline: 'Goal plan is short by 2.000.000 ₫ a month',
      detail: 'Save 2.000.000 ₫ more a month.',
      amountVnd: 2_000_000,
      impactVnd: 24_000_000,
      dueDate: undefined,
      route: '/accounts/goals',
    }),
    needsYouItem({
      id: 'rec-1a2b3c4d',
      kind: 'recommendation',
      status: 'info',
      headline: 'Cook at home twice a week',
      detail: 'Dining is the largest flexible category.',
      amountVnd: null,
      impactVnd: 1_200_000,
      dueDate: undefined,
      count: undefined,
      route: '/recommendations?page=%2Fbudget',
      recommendationId: 'rec-1a2b3c4d',
    }),
  ],
  unavailable: [],
  ranking: 'Severity first.',
  ...overrides,
});
