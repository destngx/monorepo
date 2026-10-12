import { describe, expect, it, vi } from 'vitest';
import { ApiError } from '$lib/api/client';
import type { Transaction } from '$lib/api/types';
import { nextEntry, prefillFromURL, recategorise, toInput } from './entry';

const row = (id: string, category = 'Food'): Transaction =>
  ({
    id,
    accountName: 'Cash',
    date: '2026-10-13',
    referenceNumber: null,
    payee: `Payee ${id}`,
    tags: ['trip'],
    memo: null,
    category,
    cleared: true,
    payment: 50_000,
    deposit: null,
    etag: `etag-${id}`,
  }) as Transaction;

describe('nextEntry', () => {
  it('keeps the account, date and cleared flag and clears the rest', () => {
    const next = nextEntry({
      accountName: 'Cash',
      date: '2026-10-12',
      payee: 'Coffee',
      category: 'Food',
      tags: ['x'],
      memo: 'm',
      cleared: false,
      payment: 40_000,
    });
    expect(next).toEqual({ accountName: 'Cash', date: '2026-10-12', cleared: false });
  });
});

describe('prefillFromURL', () => {
  const accounts = ['Cash', 'TCB'];
  const categories = ['Food', 'Balance Adjustment'];

  it('ignores URLs that do not ask for a new transaction', () => {
    expect(prefillFromURL(new URLSearchParams('account=Cash'), accounts, categories)).toBeNull();
  });

  it('prefills known values', () => {
    const params = new URLSearchParams({
      new: '1',
      account: 'TCB',
      payee: 'Balance adjustment',
      category: 'balance adjustment',
    });
    expect(prefillFromURL(params, accounts, categories)).toEqual({
      accountName: 'TCB',
      payee: 'Balance adjustment',
      category: 'Balance Adjustment',
    });
  });

  it('drops unknown accounts and categories', () => {
    const params = new URLSearchParams({ new: '1', account: 'Nope', category: 'Nope', payee: ' ' });
    expect(prefillFromURL(params, accounts, categories)).toEqual({});
  });
});

describe('toInput', () => {
  it('maps nulls to absent fields', () => {
    expect(toInput(row('1'))).toEqual({
      accountName: 'Cash',
      date: '2026-10-13',
      referenceNumber: undefined,
      payee: 'Payee 1',
      tags: ['trip'],
      memo: undefined,
      category: 'Food',
      cleared: true,
      payment: 50_000,
      deposit: undefined,
    });
  });
});

describe('recategorise', () => {
  it('patches rows one by one with their etag and reports each failure', async () => {
    const calls: string[] = [];
    const update = vi.fn((id: string, etag: string, input: { category: string }) => {
      calls.push(`${id}:${etag}:${input.category}`);
      if (id === '2') return Promise.reject(new ApiError(409, 'conflict', 'CONFLICT'));
      if (id === '3') return Promise.reject(new Error('network down'));
      return Promise.resolve();
    });

    const result = await recategorise([row('1'), row('2'), row('3'), row('4', 'Fun')], 'Fun', update);

    expect(calls).toEqual(['1:etag-1:Fun', '2:etag-2:Fun', '3:etag-3:Fun']);
    expect(result.updated).toEqual(['1', '4']);
    expect(result.failed.map(({ id, conflict }) => ({ id, conflict }))).toEqual([
      { id: '2', conflict: true },
      { id: '3', conflict: false },
    ]);
    expect(result.failed[0].message).toMatch(/Changed in the sheet/);
    expect(result.failed[1].message).toBe('network down');
  });
});
