import { buildPath } from '$lib/api/client';
import type { CaptureDrafts, CaptureMaterialEntry, CaptureResult, QuickCapture } from '$lib/api/types';
import { formatDateTime } from '$lib/format';
import type { InboxItem } from './materials';
import { editableInput, rowKey, toReviewRows, type ReviewRow } from './review';

// Quick capture drafts (WM-CAP-19): the engine read the phone's screenshots
// in the background; the review screen approves or rejects what it found.

/** Drafts as a capture result; each suggestion takes its draft's id. */
export function draftResult({ drafts, balances }: CaptureDrafts): CaptureResult {
  const suggestions = drafts.map((draft) => ({ ...draft.suggestion, id: draft.id }));
  return { suggestions, unreadable: [], balances };
}

/**
 * Review rows showing each draft as last saved, oldest transaction first,
 * so approved rows reach the sheet in the order they happened. The
 * suggestion keeps what the engine read.
 */
export function toDraftRows(drafts: CaptureDrafts): ReviewRow[] {
  const byId = new Map(drafts.drafts.map((draft) => [draft.id, draft]));
  const rows = toReviewRows(draftResult(drafts)).map((row) => {
    const edit = byId.get(row.suggestion.id)?.edit;
    const shown = edit
      ? { ...row, tx: editableInput(edit.transaction), categorySource: edit.categorySource, note: edit.note ?? '' }
      : row;
    return { ...shown, savedKey: rowKey(shown) };
  });
  const keys = new Map(rows.map((row) => [row.suggestion.id, draftOrder(row, byId.get(row.suggestion.id)?.createdAt)]));
  return rows.sort((a, b) => keys.get(a.suggestion.id)!.localeCompare(keys.get(b.suggestion.id)!));
}

/** Sort key: the row's date as saved, the time read from the material, then capture time and id. */
function draftOrder(row: ReviewRow, createdAt = ''): string {
  const time = row.suggestion.occurredAt.slice(11).trim() || '99:99';
  return `${row.tx.date} ${time.padStart(5, '0')} ${createdAt} ${row.suggestion.id}`;
}

/**
 * Applies a reload: drafts still pending keep the user's edits, new drafts
 * join, and drafts decided elsewhere go; the list stays in date order, by
 * the dates as saved, so a row being edited does not jump. Rows approved
 * here stay, locked, so the rules they teach can still be saved (WM-CAP-11).
 */
export function mergeDraftRows(current: ReviewRow[], fresh: ReviewRow[]): ReviewRow[] {
  const order = new Map(fresh.map((row, index) => [row.suggestion.id, index]));
  const kept = current.filter((row) => row.createdId || order.has(row.suggestion.id));
  const keptIds = new Set(kept.map((row) => row.suggestion.id));
  const merged = [...kept, ...fresh.filter((row) => !keptIds.has(row.suggestion.id))];
  // Rows gone from the reload (approved here) keep their place before the next fresh row.
  let last = -1;
  const position = new Map<string, number>();
  for (const row of current) {
    const index = order.get(row.suggestion.id);
    if (index !== undefined) last = index;
    else position.set(row.suggestion.id, last + 0.5);
  }
  const rank = (row: ReviewRow) => order.get(row.suggestion.id) ?? position.get(row.suggestion.id) ?? -1;
  return merged.sort((a, b) => rank(a) - rank(b));
}

/** Materials as review items: screenshots load from the engine, never cached. */
export function draftItems(materials: CaptureMaterialEntry[]): InboxItem[] {
  return materials.map((material) => ({
    id: material.id,
    kind: material.kind,
    name: material.kind === 'image' ? `Screenshot ${formatDateTime(material.createdAt)}` : 'Note',
    label: material.kind === 'image' ? 'Screenshot' : 'Note',
    size: 0,
    mediaType: material.mediaType,
    text: material.text,
    previewUrl:
      material.kind === 'image' ? buildPath(`/capture/materials/${encodeURIComponent(material.id)}`) : undefined,
  }));
}

export function captureStatus(capture: QuickCapture): string {
  switch (capture.status) {
    case 'queued':
      if (capture.attempts > 0 && capture.nextAttemptAt) {
        const reason = capture.message ? ` (${capture.message})` : '';
        return `Attempt ${capture.attempts} failed${reason}, retrying at ${formatDateTime(capture.nextAttemptAt)}`;
      }
      return 'Waiting to be read';
    case 'processing':
      return 'Reading…';
    case 'failed':
      return `Failed: ${capture.message || 'unknown error'}`;
    case 'done':
      if (capture.newDrafts > 0) return `${capture.newDrafts} new draft${capture.newDrafts === 1 ? '' : 's'}`;
      return capture.message || 'Nothing new.';
  }
}
