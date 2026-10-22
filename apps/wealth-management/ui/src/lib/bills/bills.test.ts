import { describe, expect, it } from 'vitest';
import type { AccountMinimumBalance, BillItem, BillOccurrence, RecurringSuggestion, SafeToSpend } from '$lib/api/types';
import {
  billStatus,
  minimumRows,
  minimumStatus,
  confidenceStatus,
  dueLabel,
  emptyRecurringForm,
  groupItems,
  recurringFormFor,
  recurringFormFromSuggestion,
  recurringInputFrom,
  safeToSpendBadge,
  scheduleLabel,
  splitCalendar,
  suggestionEvidence,
} from './bills';

const occurrence = (overrides: Partial<BillOccurrence>): BillOccurrence => ({
  name: 'Rent',
  kind: 'bill',
  source: 'recurring',
  expectedDate: '2026-10-01',
  expectedAmount: 8_000_000,
  status: 'due',
  daysUntil: 10,
  paidAmount: 0,
  remaining: 8_000_000,
  ...overrides,
});

const suggestion: RecurringSuggestion = {
  key: 'out|water co|utilities',
  name: 'Water Co',
  kind: 'bill',
  amount: 200_000,
  frequency: 'monthly',
  day: 5,
  account: 'Main Bank',
  category: 'Utilities',
  match: 'Water Co',
  lastSeen: '2026-09',
  seen: 5,
  checked: 6,
  amountSpread: 4,
  daySpread: 2,
  confidence: 'high',
};

describe('safe to spend badge', () => {
  const safe = (overrides: Partial<SafeToSpend>) =>
    ({ safeToSpend: 0, shortBy: 0, tightBy: 0, ...overrides }) as Pick<
      SafeToSpend,
      'safeToSpend' | 'shortBy' | 'tightBy'
    >;

  it('is short when committed money is not covered, tight when essentials are not, covered when money is free', () => {
    expect(safeToSpendBadge(safe({ shortBy: 1, tightBy: 5 }))).toEqual({ tone: 'serious', label: 'Short' });
    expect(safeToSpendBadge(safe({ tightBy: 5 }))).toEqual({ tone: 'warning', label: 'Tight' });
    expect(safeToSpendBadge(safe({ safeToSpend: 5 }))).toEqual({ tone: 'good', label: 'Covered' });
    expect(safeToSpendBadge(safe({}))).toEqual({ tone: 'warning', label: 'Nothing free' });
  });
});

describe('bill status', () => {
  it('labels every status with a tone, income in its own words', () => {
    expect(billStatus(occurrence({ status: 'paid' }))).toEqual({ tone: 'good', label: 'Paid' });
    expect(billStatus(occurrence({ status: 'paid', kind: 'income' }))).toEqual({ tone: 'good', label: 'Received' });
    expect(billStatus(occurrence({ status: 'overdue' }))).toEqual({ tone: 'serious', label: 'Overdue' });
    expect(billStatus(occurrence({ status: 'overdue', kind: 'income' }))).toEqual({ tone: 'warning', label: 'Late' });
    expect(billStatus(occurrence({ daysUntil: 2 }))).toEqual({ tone: 'warning', label: 'Due soon' });
    expect(billStatus(occurrence({ daysUntil: 9 }))).toEqual({ tone: 'info', label: 'Due' });
    expect(confidenceStatus('low').tone).toBe('warning');
  });

  it('says when in words', () => {
    expect(dueLabel(0)).toBe('Today');
    expect(dueLabel(1)).toBe('Tomorrow');
    expect(dueLabel(5)).toBe('In 5 days');
    expect(dueLabel(-3)).toBe('3 days ago');
  });
});

describe('schedule and evidence', () => {
  it('describes each frequency', () => {
    expect(scheduleLabel({ kind: 'bill', frequency: 'monthly', day: 5 })).toBe('Monthly, day 5');
    expect(scheduleLabel({ kind: 'insurance', frequency: 'yearly', day: 5, month: 11 })).toBe('Yearly, 5 Nov');
    expect(scheduleLabel({ kind: 'bill', frequency: 'quarterly', day: 15, month: 2 })).toBe(
      'Quarterly from Feb, day 15',
    );
    expect(scheduleLabel({ kind: 'card', frequency: 'monthly', day: 0 })).toContain('29th');
  });

  it('explains a suggestion', () => {
    expect(suggestionEvidence(suggestion)).toBe(
      'Seen in 5 of the last 6 months, amounts within 4%, within 2 days of day 5',
    );
    expect(suggestionEvidence({ ...suggestion, frequency: 'yearly', seen: 3, amountSpread: 0 })).toBe(
      'Seen in 3 years in a row, same amount',
    );
  });
});

describe('calendar and items', () => {
  it('splits this month so far from what is coming, overdue bills kept in view first', () => {
    const { earlier, upcoming } = splitCalendar(
      [
        occurrence({ name: 'Rent', expectedDate: '2026-10-01', status: 'paid' }),
        occurrence({ name: 'Internet', expectedDate: '2026-10-10', status: 'overdue' }),
        occurrence({ name: 'Phone', expectedDate: '2026-10-17' }),
      ],
      '2026-10-17',
    );
    expect(earlier.map((o) => o.name)).toEqual(['Rent']);
    expect(upcoming.map((o) => o.name)).toEqual(['Internet', 'Phone']);
  });

  it('groups active and paused items', () => {
    const item = { status: 'active' } as BillItem;
    const groups = groupItems([item, { ...item, status: 'paused' }]);
    expect(groups.active).toHaveLength(1);
    expect(groups.paused).toHaveLength(1);
  });
});

describe('recurring form', () => {
  it('builds the input from a suggestion', () => {
    const { input, error } = recurringInputFrom(recurringFormFromSuggestion(suggestion));
    expect(error).toBeUndefined();
    expect(input).toEqual({
      name: 'Water Co',
      kind: 'bill',
      amount: 200_000,
      frequency: 'monthly',
      day: 5,
      month: undefined,
      account: 'Main Bank',
      category: 'Utilities',
      match: 'Water Co',
      status: 'active',
      note: undefined,
    });
  });

  it('round-trips an item and rejects what the engine would', () => {
    const item: BillItem = {
      id: 'recurring-3',
      row: 3,
      name: 'Cover',
      kind: 'insurance',
      amount: 2_000_000,
      frequency: 'yearly',
      day: 5,
      month: 11,
      match: 'Cover Co',
      status: 'paused',
      etag: 'x',
      current: null,
      lastMatch: null,
      matchCount: 0,
    };
    expect(recurringInputFrom(recurringFormFor(item)).input?.month).toBe(11);
    const blank = emptyRecurringForm();
    expect(recurringInputFrom(blank).error).toBe('Name is required.');
    expect(recurringInputFrom({ ...blank, name: 'Gym' }).error).toBe('Amount is required.');
    expect(recurringInputFrom({ ...blank, name: 'Gym', amount: 1, day: 40 }).error).toContain('Day');
    expect(recurringInputFrom({ ...blank, name: 'Gym', amount: 1, day: 4, frequency: 'yearly' }).error).toContain(
      'month',
    );
    expect(recurringInputFrom({ ...blank, name: 'Gym', amount: 1, day: 4 }).error).toContain('match text');
    const card = recurringInputFrom({ ...blank, name: 'Card A', kind: 'card', amount: 5, account: 'Card Account' });
    expect(card.input).toMatchObject({ kind: 'card', amount: undefined, day: undefined, account: 'Card Account' });
  });
});

describe('minimum balances (D35)', () => {
  const track = (account: string, status: AccountMinimumBalance['status'], daysLeft = 15) =>
    ({ account, status, daysLeft }) as AccountMinimumBalance;

  it('Given a status When labelled Then at risk is a warning, serious in the last week, and on track is good', () => {
    expect(minimumStatus(track('A', 'on_track'))).toEqual({ tone: 'good', label: 'On track' });
    expect(minimumStatus(track('A', 'at_risk', 8))).toEqual({ tone: 'warning', label: 'At risk' });
    expect(minimumStatus(track('A', 'at_risk', 7))).toEqual({ tone: 'serious', label: 'At risk' });
  });

  it('Given ready, not-ready and crypto accounts When listing rows Then minimums come first and crypto is never offered', () => {
    const safe = {
      minimumBalances: [track('Bank B', 'on_track')],
      readyAccounts: [{ name: 'Bank A' }, { name: 'bank b' }],
      excludedAccounts: [
        { name: 'Broker C', reason: 'not_ready' },
        { name: 'Coins D', reason: 'crypto' },
      ],
    } as unknown as SafeToSpend;
    expect(minimumRows(safe).map((row) => [row.account, row.track !== null])).toEqual([
      ['Bank B', true],
      ['Bank A', false],
      ['Broker C', false],
    ]);
  });
});

describe('grace days (D42)', () => {
  it('Given a bill past its date within its grace When labelled Then it is late but not urgent', () => {
    expect(billStatus(occurrence({ status: 'due', daysUntil: -3, graceUntil: '2026-10-15' }))).toEqual({
      tone: 'info',
      label: 'Late, in grace',
    });
    expect(billStatus(occurrence({ status: 'overdue', daysUntil: -20 })).label).toBe('Overdue');
  });

  it('Given a late bill within its grace When splitting the calendar Then it stays in view after the overdue ones', () => {
    const { earlier, upcoming } = splitCalendar(
      [
        occurrence({
          name: 'Rent',
          expectedDate: '2026-10-01',
          status: 'due',
          daysUntil: -16,
          graceUntil: '2026-10-15',
        }),
        occurrence({ name: 'Internet', expectedDate: '2026-10-10', status: 'overdue' }),
        occurrence({ name: 'Water', expectedDate: '2026-10-05', status: 'paid' }),
        occurrence({ name: 'Phone', expectedDate: '2026-10-20' }),
      ],
      '2026-10-17',
    );
    expect(earlier.map((o) => o.name)).toEqual(['Water']);
    expect(upcoming.map((o) => o.name)).toEqual(['Internet', 'Rent', 'Phone']);
  });

  it('Given grace days When describing the schedule Then they are named', () => {
    expect(scheduleLabel({ kind: 'bill', frequency: 'monthly', day: 28, graceDays: 14 })).toBe(
      'Monthly, day 28, 14 days grace',
    );
    expect(scheduleLabel({ kind: 'bill', frequency: 'yearly', day: 5, month: 11, graceDays: 1 })).toBe(
      'Yearly, 5 Nov, 1 day grace',
    );
  });

  it('Given grace days on the form When building the input Then they round-trip, are checked and dropped on cards', () => {
    const rent = { ...emptyRecurringForm(), name: 'Rent', amount: 8_000_000, day: 28, match: 'Landlord' };
    expect(recurringInputFrom({ ...rent, graceDays: 14 }).input?.graceDays).toBe(14);
    expect(recurringInputFrom({ ...rent, graceDays: 0 }).input?.graceDays).toBeUndefined();
    expect(recurringInputFrom({ ...rent, graceDays: 22 }).error).toContain('Grace days');
    expect(recurringInputFrom({ ...rent, graceDays: 1.5 }).error).toContain('Grace days');
    const card = recurringInputFrom({ ...rent, kind: 'card', account: 'Card Account', graceDays: 5 });
    expect(card.input?.graceDays).toBeUndefined();
    const item: BillItem = {
      id: 'recurring-3',
      row: 3,
      name: 'Rent',
      kind: 'bill',
      amount: 8_000_000,
      frequency: 'monthly',
      day: 28,
      match: 'Landlord',
      status: 'active',
      graceDays: 14,
      etag: 'x',
      current: null,
      lastMatch: null,
      matchCount: 0,
    };
    expect(recurringFormFor(item).graceDays).toBe(14);
    expect(recurringFormFor({ ...item, graceDays: undefined }).graceDays).toBeNull();
  });
});
