import { describe, expect, it } from 'vitest';
import type { Transaction } from '$lib/api/types';
import { needsCategory, paginate, searchTransactions, sortTransactions } from './filter';

const tx = (overrides: Partial<Transaction>): Transaction => ({
  id: 'row-1',
  accountName: 'Bank',
  date: '2026-10-01',
  referenceNumber: null,
  payee: 'Shop',
  tags: [],
  memo: null,
  category: 'Food',
  cleared: true,
  payment: null,
  deposit: null,
  etag: 'e',
  ...overrides,
});

describe('searchTransactions', () => {
  const rows = [
    tx({ id: 'a', payee: 'Highlands Coffee' }),
    tx({ id: 'b', memo: 'birthday gift' }),
    tx({ id: 'c', tags: ['UNIQ Supermarket'] }),
  ];

  it('matches payee, memo and tags case-insensitively', () => {
    expect(searchTransactions(rows, 'COFFEE').map((r) => r.id)).toEqual(['a']);
    expect(searchTransactions(rows, 'gift').map((r) => r.id)).toEqual(['b']);
    expect(searchTransactions(rows, 'uniq').map((r) => r.id)).toEqual(['c']);
    expect(searchTransactions(rows, '  ')).toHaveLength(3);
  });
});

describe('sortTransactions', () => {
  it('sorts numbers with missing amounts as zero and keeps ties stable', () => {
    const rows = [tx({ id: 'a', payment: 5 }), tx({ id: 'b' }), tx({ id: 'c', payment: 5 })];
    expect(sortTransactions(rows, 'payment', 'desc').map((r) => r.id)).toEqual(['a', 'c', 'b']);
    expect(sortTransactions(rows, 'payment', 'asc').map((r) => r.id)).toEqual(['b', 'a', 'c']);
  });
});

describe('paginate', () => {
  it('clamps the page into range', () => {
    const result = paginate([1, 2, 3, 4, 5], 9, 2);
    expect(result).toEqual({ items: [5], pages: 3, page: 3 });
  });
});

describe('needsCategory', () => {
  it('matches blank, Uncategorized and unknown categories like the engine', () => {
    expect(needsCategory(tx({ category: '', categoryType: 'expense' }))).toBe(true);
    expect(needsCategory(tx({ category: ' Uncategorized ', categoryType: 'expense' }))).toBe(true);
    expect(needsCategory(tx({ category: 'Mystery' }))).toBe(true);
    expect(needsCategory(tx({ category: 'Food', categoryType: 'expense' }))).toBe(false);
    expect(needsCategory(tx({ category: 'Transfer', categoryType: 'non-budget' }))).toBe(false);
  });
});
