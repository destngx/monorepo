import { describe, expect, it } from 'vitest';
import type { SearchHit } from './api/types';
import { hitHref, hitTitle } from './search';

const hit = (over: Partial<SearchHit>): SearchHit => ({ kind: 'page', snippet: [], ...over });

describe('hitHref', () => {
  it('opens a filed page in the document viewer at that page', () => {
    expect(hitHref(hit({ documentId: 'd1', pageId: 'p1' }))).toBe('/documents/d1?page=p1');
  });
  it('opens an unfiled page in the inbox review', () => {
    expect(hitHref(hit({ itemId: 'i1', pageId: 'p1' }))).toBe('/inbox/i1');
  });
  it('opens findings in their document and visits on their own page', () => {
    expect(hitHref(hit({ kind: 'finding', documentId: 'd1' }))).toBe('/documents/d1');
    expect(hitHref(hit({ kind: 'visit', visitId: 'v1' }))).toBe('/visits/v1');
  });
});

describe('hitTitle', () => {
  it('names the document, then the visit, then the inbox', () => {
    expect(hitTitle(hit({ documentTitle: 'Blood test', documentKind: 'lab_result' }))).toBe('Blood test');
    expect(hitTitle(hit({ documentKind: 'lab_result' }))).toBe('Lab result');
    expect(hitTitle(hit({ kind: 'visit', visitTitle: '' }))).toBe('Visit');
    expect(hitTitle(hit({ itemId: 'i1' }))).toBe('Inbox item');
  });
});
