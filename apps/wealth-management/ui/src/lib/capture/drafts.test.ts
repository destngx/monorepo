import { describe, expect, it } from 'vitest';
import type { CaptureDraft, CaptureDrafts, CaptureSuggestion, QuickCapture } from '$lib/api/types';
import { captureStatus, draftItems, draftResult, mergeDraftRows, toDraftRows } from './drafts';

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
  it('lists the newest drafts first, keyed by draft id', () => {
    const result = draftResult(
      drafts([draft('drf_old', '2026-10-15T09:00:00Z'), draft('drf_new', '2026-10-16T09:00:00Z')]),
    );

    expect(result.suggestions.map((s) => s.id)).toEqual(['drf_new', 'drf_old']);
    expect(result.unreadable).toEqual([]);
  });
});

describe('mergeDraftRows', () => {
  it('keeps edits and positions of drafts still pending and appends new drafts', () => {
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
    current[0].createdId = 'tx1';

    const merged = mergeDraftRows(current, []);

    expect(merged.map((row) => row.suggestion.id)).toEqual(['drf_1']);
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
