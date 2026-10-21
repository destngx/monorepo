import type { SearchHit } from './api/types';
import { documentKindLabels } from './format';

/** Where a search hit opens, without the /ui base. */
export function hitHref(hit: SearchHit): string {
  if (hit.kind === 'visit') return `/visits/${hit.visitId}`;
  if (hit.documentId) {
    return hit.kind === 'page' && hit.pageId
      ? `/documents/${hit.documentId}?page=${hit.pageId}`
      : `/documents/${hit.documentId}`;
  }
  return `/inbox/${hit.itemId}`;
}

/** The hit's heading: the document's title or kind, else the visit or the inbox. */
export function hitTitle(hit: SearchHit): string {
  if (hit.documentTitle) return hit.documentTitle;
  if (hit.documentKind) return documentKindLabels[hit.documentKind];
  if (hit.kind === 'visit') return hit.visitTitle || 'Visit';
  return 'Inbox item';
}
