import { describe, expect, it } from 'vitest';
import type { FillResult } from '$lib/api/types';
import { applyFill, asNumber, asText, filledValues, fillSummary, notesFor, sameValue, takesImages } from './fill';

const result = (overrides: Partial<FillResult> = {}): FillResult => ({
  form: 'goal',
  values: {},
  filled: [],
  notes: [],
  ...overrides,
});

describe('applyFill', () => {
  it('merges only the keys the form has', () => {
    const form = { name: '', target: null as number | null, note: 'kept' };
    const next = applyFill(form, { name: 'Trip', target: 50_000_000, invented: 'x' });
    expect(next).toEqual({ name: 'Trip', target: 50_000_000, note: 'kept' });
    expect(form.name).toBe('');
  });

  it('converts values the form holds differently', () => {
    const next = applyFill(
      { amount: '', tags: '' },
      { amount: 260_000, tags: ['work', 'trip'] },
      { amount: asText, tags: asText },
    );
    expect(next).toEqual({ amount: '260000', tags: 'work, trip' });
  });
});

describe('marks and notes', () => {
  it('marks only the fields the answer changed', () => {
    const answer = result({ values: { name: 'Trip', targetDate: '2028-06' }, filled: ['targetDate'] });
    expect(filledValues(answer)).toEqual({ targetDate: '2028-06' });
  });

  it('compares numbers, text and lists as the form holds them', () => {
    expect(sameValue(50_000_000, 50_000_000)).toBe(true);
    expect(sameValue(['work', 'trip'], 'work,trip')).toBe(true);
    expect(sameValue('Trip ', 'Trip')).toBe(true);
    expect(sameValue(5, 6)).toBe(false);
    expect(sameValue(null, '')).toBe(true);
  });

  it('picks the notes of a field or row', () => {
    const notes = [
      { field: 'category', row: 1, message: 'No category matches "Pets".' },
      { field: 'name', message: 'Name is still needed.' },
      { message: 'Which account pays it?' },
    ];
    expect(notesFor(notes, 'category', 1)).toHaveLength(1);
    expect(notesFor(notes, 'name')).toHaveLength(1);
    expect(notesFor(notes)).toEqual([{ message: 'Which account pays it?' }]);
  });
});

describe('fillSummary', () => {
  it('counts fields and rows', () => {
    expect(fillSummary(result({ filled: ['name', 'target'] }))).toBe('Filled 2 fields. Check them, then save.');
    expect(fillSummary(result({ filled: ['rows'], rows: [{}, {}] }), 'limit')).toBe(
      'Filled 2 limits. Check them, then save.',
    );
    expect(fillSummary(result())).toBe('Nothing to fill from that.');
  });
});

describe('helpers', () => {
  it('reads numbers and text', () => {
    expect(asNumber(5)).toBe(5);
    expect(asNumber('5')).toBeNull();
    expect(asText(null)).toBe('');
    expect(asText(12)).toBe('12');
  });

  it('knows which forms take images', () => {
    expect(takesImages('recurring')).toBe(true);
    expect(takesImages('budget-move')).toBe(false);
  });
});
