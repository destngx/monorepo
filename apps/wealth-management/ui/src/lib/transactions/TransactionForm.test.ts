import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ fill: vi.fn(), suggestCategory: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));

import type { Account, Category, Transaction } from '$lib/api/types';
import TransactionForm from './TransactionForm.svelte';

const accounts = [
  { name: 'Cash', type: 'cash' },
  { name: 'Wallet A', type: 'active use' },
] as Account[];
const categories = [
  { name: 'Food', type: 'expense' },
  { name: 'Salary', type: 'income' },
] as Category[];

describe('TransactionForm: Describe it (WM-FILL-6)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('fills the row, adds tags to the ones it has and moves the amount to the column the answer names', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'transaction',
      values: { accountName: 'Wallet A', payee: 'Coffee shop', category: 'Food', tags: ['work'], payment: 45_000 },
      filled: ['accountName', 'payee', 'category', 'tags', 'payment'],
      notes: [],
    });
    const onsubmit = vi.fn();
    const initial = {
      id: 't1',
      accountName: 'Cash',
      date: '2026-10-22',
      referenceNumber: null,
      payee: 'Old',
      tags: ['trip'],
      memo: null,
      category: 'Salary',
      cleared: true,
      payment: null,
      deposit: 45_000,
      etag: 'e1',
    } as Transaction;
    render(TransactionForm, { accounts, categories, tags: ['trip', 'work'], initial, onsubmit, oncancel: vi.fn() });

    await fireEvent.input(screen.getByRole('textbox', { name: 'Describe it' }), {
      target: { value: 'it was coffee 45k from the wallet, for work' },
    });
    await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));
    await screen.findByText('Filled 5 fields. Check them, then save.');

    const [request] = endpoints.fill.mock.calls[0] as [{ form: string; current: Record<string, unknown> }];
    expect(request.form).toBe('transaction');
    expect(request.current).toMatchObject({ payee: 'Old', deposit: 45_000 });
    await fireEvent.submit(document.getElementById('transaction-form')!);
    expect(onsubmit).toHaveBeenCalledWith(
      expect.objectContaining({
        accountName: 'Wallet A',
        payee: 'Coffee shop',
        category: 'Food',
        tags: ['trip', 'work'],
        payment: 45_000,
        deposit: undefined,
      }),
      false,
    );
  });
});
