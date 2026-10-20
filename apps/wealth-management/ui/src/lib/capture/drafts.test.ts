import { describe, expect, it } from 'vitest';
import type { CaptureDraft, CaptureDrafts, CaptureSuggestion, QuickCapture } from '$lib/api/types';
import { captureStatus, draftItems, draftResult, mergeDraftRows, toDraftRows } from './drafts';
import { isRowDirty } from './review';

function suggestion(overrides: Partial<CaptureSuggestion> = {}): CaptureSuggestion {
  return {
    id: 's1',
    selected: true,
    transaction: {
      accountName: 'Sacombank',
      date: '16/10/2026',
      payee: 'Pho 24',
      category: 'Food',
      tags: [],
      cleared: true,
      payment: 50000,
    },
    direction: 'out',
    amount: 50000,
    currency: 'VND',
    occurredAt: '2026-10-16 08:15',
    confidence: 0.9,
    lowConfidence: [],
    evidence: [{ materialId: 'cap_a-image' }],
    ...overrides,
  };
}

function draft(id: string, createdAt: string, overrides: Partial<CaptureSuggestion> = {}): CaptureDraft {
  return { id, status: 'pending', createdAt, captureIds: ['cap_a'], suggestion: suggestion(overrides) };
}

const drafts = (list: CaptureDraft[]): CaptureDrafts => ({ drafts: list, materials: [], balances: [] });

describe('draftResult', () => {
  it('keys each suggestion by its draft id', () => {
    const result = draftResult(drafts([draft('drf_1', '2026-10-15T09:00:00Z')]));

    expect(result.suggestions.map((s) => s.id)).toEqual(['drf_1']);
    expect(result.unreadable).toEqual([]);
  });
});

describe('toDraftRows order', () => {
  const on = (date: string, occurredAt: string) => ({
    occurredAt,
    transaction: { ...suggestion().transaction, date },
  });

  it('lists the oldest transaction first, by date then time, whenever it was captured', () => {
    const rows = toDraftRows(
      drafts([
        draft('drf_late', '2026-10-16T07:00:00Z', on('16/10/2026', '2026-10-16 18:00')),
        draft('drf_old', '2026-10-17T07:00:00Z', on('14/10/2026', '2026-10-14')),
        draft('drf_morning', '2026-10-15T07:00:00Z', on('16/10/2026', '2026-10-16 08:15')),
      ]),
    );

    expect(rows.map((row) => row.suggestion.id)).toEqual(['drf_old', 'drf_morning', 'drf_late']);
  });

  it('orders by the date as saved', () => {
    const moved: CaptureDraft = {
      ...draft('drf_1', '2026-10-16T07:00:00Z', on('16/10/2026', '2026-10-16 08:15')),
      edit: { transaction: { ...suggestion().transaction, date: '2026-10-01' } },
    };
    const rows = toDraftRows(drafts([draft('drf_2', '2026-10-16T07:00:00Z', on('10/10/2026', '2026-10-10')), moved]));

    expect(rows.map((row) => row.suggestion.id)).toEqual(['drf_1', 'drf_2']);
  });
});

describe('toDraftRows', () => {
  it('shows a saved edit in place of what the engine read, as saved', () => {
    const edited: CaptureDraft = {
      ...draft('drf_1', '2026-10-16T09:00:00Z', { categorySource: 'history' }),
      edit: {
        transaction: { ...suggestion().transaction, date: '2026-10-17', payee: 'Grab', category: 'Transport' },
        categorySource: 'ai',
        note: 'ride to the office',
      },
    };

    const [row] = toDraftRows(drafts([edited]));

    expect([row.tx.payee, row.tx.category, row.tx.date, row.categorySource, row.note]).toEqual([
      'Grab',
      'Transport',
      '2026-10-17',
      'ai',
      'ride to the office',
    ]);
    expect(row.suggestion.transaction.payee).toBe('Pho 24');
    expect(isRowDirty(row)).toBe(false);
    row.tx.payee = 'Be';
    expect(isRowDirty(row)).toBe(true);
  });

  it('starts with an empty note when the draft has none', () => {
    const [row] = toDraftRows(drafts([draft('drf_1', '2026-10-16T09:00:00Z', {})]));

    expect(row.note).toBe('');
  });
});

describe('mergeDraftRows', () => {
  it('keeps edits of drafts still pending and places new drafts in date order', () => {
    const current = toDraftRows(drafts([draft('drf_1', '2026-10-16T09:00:00Z')]));
    current[0].tx.payee = 'Edited';

    const merged = mergeDraftRows(
      current,
      toDraftRows(drafts([draft('drf_2', '2026-10-16T10:00:00Z'), draft('drf_1', '2026-10-16T09:00:00Z')])),
    );

    expect(merged.map((row) => [row.suggestion.id, row.tx.payee])).toEqual([
      ['drf_1', 'Edited'],
      ['drf_2', 'Pho 24'],
    ]);
  });

  it('keeps approved rows so their rules can still be learned, and drops vanished ones', () => {
    const current = toDraftRows(
      drafts([draft('drf_1', '2026-10-16T09:00:00Z'), draft('drf_2', '2026-10-16T08:00:00Z')]),
    );
    current.find((row) => row.suggestion.id === 'drf_1')!.createdId = 'tx1';

    const merged = mergeDraftRows(current, []);

    expect(merged.map((row) => row.suggestion.id)).toEqual(['drf_1']);
  });

  it('keeps an approved row in its place among the reloaded rows', () => {
    const at = (occurredAt: string) => ({ occurredAt });
    const list = [
      draft('drf_a', '2026-10-16T09:00:00Z', at('2026-10-16 08:00')),
      draft('drf_b', '2026-10-16T09:00:00Z', at('2026-10-16 09:00')),
      draft('drf_c', '2026-10-16T09:00:00Z', at('2026-10-16 10:00')),
    ];
    const current = toDraftRows(drafts(list));
    current[1].createdId = 'tx1';

    const merged = mergeDraftRows(current, toDraftRows(drafts([list[2], list[0]])));

    expect(merged.map((row) => row.suggestion.id)).toEqual(['drf_a', 'drf_b', 'drf_c']);
  });
});

describe('draftItems', () => {
  it('points screenshots at the engine and keeps notes as text', () => {
    const items = draftItems([
      {
        id: 'cap_a-image',
        captureId: 'cap_a',
        kind: 'image',
        name: 'screenshot',
        mediaType: 'image/png',
        createdAt: '2026-10-16T09:00:00Z',
      },
      { id: 'cap_a-note', captureId: 'cap_a', kind: 'text', name: 'note', text: 'team lunch', createdAt: '' },
    ]);

    expect(items[0].previewUrl).toBe('/api/capture/materials/cap_a-image');
    expect(items[1]).toMatchObject({ kind: 'text', text: 'team lunch', previewUrl: undefined });
  });
});

describe('captureStatus', () => {
  const capture = (overrides: Partial<QuickCapture>): QuickCapture => ({
    id: 'cap_a',
    createdAt: '2026-10-16T09:00:00Z',
    status: 'queued',
    hasImage: true,
    attempts: 0,
    newDrafts: 0,
    ...overrides,
  });

  it('describes each state', () => {
    expect(captureStatus(capture({}))).toBe('Waiting to be read');
    expect(captureStatus(capture({ status: 'processing' }))).toBe('Reading…');
    expect(captureStatus(capture({ status: 'done', newDrafts: 2 }))).toBe('2 new drafts');
    expect(captureStatus(capture({ status: 'done', newDrafts: 1 }))).toBe('1 new draft');
    expect(captureStatus(capture({ status: 'done', message: 'Nothing found.' }))).toBe('Nothing found.');
    expect(captureStatus(capture({ status: 'failed', message: 'provider down' }))).toBe('Failed: provider down');
  });

  it('says when a failed attempt will be retried', () => {
    const status = captureStatus(
      capture({ attempts: 2, nextAttemptAt: '2026-10-16T09:05:00Z', message: 'provider down' }),
    );

    expect(status).toMatch(/^Attempt 2 failed \(provider down\), retrying at /);
  });
});
