import { describe, expect, it } from 'vitest';
import type { CloseItem, MonthlyClose } from '$lib/api/types';
import { shiftMonth } from '$lib/format';
import { CLOSE_STATE, closeFigures, closeHeadline, closeLink, closeMonths } from './close';

const item = (overrides: Partial<CloseItem>): CloseItem => ({
  key: 'drafts',
  title: 'Approve capture drafts',
  state: 'done',
  detail: '',
  count: null,
  amount: null,
  target: null,
  ...overrides,
});

const close = (done: number, checked: number): MonthlyClose => ({
  month: '2026-10',
  asOf: '2026-10-31',
  monthEnded: true,
  canWriteSnapshot: true,
  items: [],
  done,
  checked,
  snapshot: null,
});

describe('monthly close', () => {
  it('links every item to the page where the user acts', () => {
    expect(closeLink('budget', '2026-10')).toEqual({ href: '/budget?month=2026-10', label: 'Open the budget' });
    expect(closeLink('drafts', '2026-10')?.href).toBe('/transactions/drafts');
    expect(closeLink('recommendations', '2026-10')?.href).toBe('/recommendations');
  });

  it('maps each state to a status tone with a label', () => {
    expect(CLOSE_STATE.done).toEqual({ tone: 'good', label: 'Done' });
    expect(CLOSE_STATE.action.tone).toBe('warning');
    expect(CLOSE_STATE.unavailable.tone).toBe('neutral');
  });

  it('shows only the figures that apply', () => {
    expect(closeFigures(item({ key: 'budget', count: 0, amount: 0, target: 20_000_000 }))).toEqual([
      { label: 'Total limit', value: 20_000_000 },
    ]);
    expect(closeFigures(item({ key: 'budget', count: 2, amount: 3_000_000, target: 20_000_000 }))).toHaveLength(2);
    expect(closeFigures(item({ key: 'goals', amount: 12_000_000, target: 15_000_000 }))).toEqual([
      { label: 'Saved this month', value: 12_000_000 },
      { label: 'Goals need a month', value: 15_000_000 },
    ]);
    expect(closeFigures(item({ key: 'goals' }))).toEqual([]);
    expect(closeFigures(item({ key: 'drafts', count: 3 }))).toEqual([]);
    expect(closeFigures(item({ key: 'recommendations', count: 2, amount: 0 }))).toEqual([]);
  });

  it('heads the page with the checks done', () => {
    expect(closeHeadline(close(4, 6))).toEqual({
      text: '4 of 6 checks done for Oct 2026; 2 need you.',
      tone: 'warning',
    });
    expect(closeHeadline(close(5, 6)).text).toContain('1 needs you');
    expect(closeHeadline(close(6, 6))).toEqual({ text: 'Oct 2026 is closed: all 6 checks are done.', tone: 'good' });
  });

  it('offers the close month and the two before it', () => {
    expect(closeMonths('2026-01', shiftMonth)).toEqual(['2025-11', '2025-12', '2026-01']);
  });
});
