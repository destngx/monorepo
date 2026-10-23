import type { Page } from '$lib/api/types';

/** A page as people see it: its number in the document and the title the AI gave it, `Page 2 · Siêu âm ổ bụng`. */
export function pageName(number: number, title?: string): string {
  return title ? `Page ${number} · ${title}` : `Page ${number}`;
}

/** Looks up a page of a document by id: its 1-based number across the document's files and its name. */
export function pageLookup(pages: Page[]): (id: string | undefined) => { number: number; name: string } | undefined {
  const byId = new Map(pages.map((p, i) => [p.id, { number: i + 1, name: pageName(i + 1, p.title) }]));
  return (id) => (id ? byId.get(id) : undefined);
}

/** The document viewer's Pages tab, opened at one page. */
export function pageHref(base: string, documentId: string, pageId?: string): string {
  const query = new URLSearchParams({ tab: 'pages' });
  if (pageId) query.set('page', pageId);
  return `${base}/documents/${documentId}?${query}`;
}
