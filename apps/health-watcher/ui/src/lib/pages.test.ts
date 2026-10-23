import { describe, expect, it } from 'vitest';
import type { Page } from '$lib/api/types';
import { pageHref, pageLookup, pageName } from './pages';

const page = (id: string, title?: string) => ({ id, title }) as Page;

describe('pageName', () => {
  it('adds the title when the page has one', () => {
    expect(pageName(2, 'Siêu âm ổ bụng')).toBe('Page 2 · Siêu âm ổ bụng');
    expect(pageName(1)).toBe('Page 1');
    expect(pageName(1, '')).toBe('Page 1');
  });
});

describe('pageLookup', () => {
  it('numbers pages in document order and finds them by id', () => {
    const find = pageLookup([page('a'), page('b', 'Tổng phân tích tế bào máu')]);
    expect(find('b')).toEqual({ number: 2, name: 'Page 2 · Tổng phân tích tế bào máu' });
    expect(find('a')).toEqual({ number: 1, name: 'Page 1' });
    expect(find('z')).toBeUndefined();
    expect(find(undefined)).toBeUndefined();
  });
});

describe('pageHref', () => {
  it('opens the Pages tab, at a page when given', () => {
    expect(pageHref('/ui', 'd1', 'p2')).toBe('/ui/documents/d1?tab=pages&page=p2');
    expect(pageHref('', 'd1')).toBe('/documents/d1?tab=pages');
  });
});
