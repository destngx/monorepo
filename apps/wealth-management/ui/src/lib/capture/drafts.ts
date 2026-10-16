import { buildPath } from '$lib/api/client';
import type { CaptureDrafts, CaptureMaterialEntry, CaptureResult, QuickCapture } from '$lib/api/types';
import { formatDateTime } from '$lib/format';
import type { InboxItem } from './materials';
import { toReviewRows, type ReviewRow } from './review';

// Quick capture drafts (WM-CAP-19): the engine read the phone's screenshots
// in the background; the review screen approves or rejects what it found.

/** Drafts as a capture result, newest first; each suggestion takes its draft's id. */
export function draftResult({ drafts, balances }: CaptureDrafts): CaptureResult {
  const suggestions = [...drafts]
    .sort((a, b) => b.createdAt.localeCompare(a.createdAt) || b.id.localeCompare(a.id))
    .map((draft) => ({ ...draft.suggestion, id: draft.id }));
  return { suggestions, unreadable: [], balances };
}

export function toDraftRows(drafts: CaptureDrafts): ReviewRow[] {
  return toReviewRows(draftResult(drafts));
}

/**
 * Applies a reload: drafts still pending keep the user's edits and their
 * place (a commit awaiting retry refers to rows by position), new drafts
 * are appended, and drafts decided elsewhere go. Rows approved here stay,
 * locked, so the rules they teach can still be saved (WM-CAP-11).
 */
export function mergeDraftRows(current: ReviewRow[], fresh: ReviewRow[]): ReviewRow[] {
  const freshIds = new Set(fresh.map((row) => row.suggestion.id));
  const kept = current.filter((row) => row.createdId || freshIds.has(row.suggestion.id));
  const keptIds = new Set(kept.map((row) => row.suggestion.id));
  return [...kept, ...fresh.filter((row) => !keptIds.has(row.suggestion.id))];
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
