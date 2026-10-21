import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Account, Category, Transaction } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          return new Promise(() => undefined);
        },
    },
  ),
}));
vi.mock('$app/navigation', () => ({ goto: vi.fn(), invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$app/state', () => ({ page: { url: new URL('http://127.0.0.1:5173/ui/transactions') } }));

import TransactionsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented rows.
const tx = (id: string, payee: string, payment: number): Transaction => ({
  id,
  accountName: 'Everyday',
  date: '2026-10-10',
  referenceNumber: null,
  payee,
  tags: [],
  memo: null,
  category: 'Groceries',
  categoryType: 'expense',
  cleared: true,
  payment,
  deposit: null,
  etag: `etag-${id}`,
});
const rows = [tx('t1', 'Corner Market', 734_000), tx('t2', 'Bus Card', 52_000)];
const accounts = [{ name: 'Everyday', type: 'bank', currency: 'VND' } as Account];
const categories: Category[] = [
  { name: 'Groceries', type: 'expense' },
  { name: 'Transport', type: 'expense' },
];

const pending = () => ({ engine: never(), transactions: never(), options: never(), cardConfig: never() });

describe('Transactions page', () => {
  beforeAll(() => {
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
  });
  afterEach(() => {
    mask.hide();
    calls.length = 0;
  });

  it('Given nothing has answered When the page renders Then the header, search and Add are usable at once', () => {
    render(TransactionsPage, { data: pending() as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Transactions' })).toBeTruthy();
    expect(screen.getByText('Counting transactions…')).toBeTruthy();
    expect(screen.getByRole('searchbox', { name: 'Search transactions' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Add' }).hasAttribute('disabled')).toBe(false);
    expect(screen.getByRole('link', { name: /Drafts/ }).getAttribute('href')).toBe('/ui/transactions/drafts');
    expect(screen.getByText('Loading transactions…')).toBeTruthy();
  });

  it('Given the ledger fails When it settles Then the table shows the error with Retry and the shell stays', async () => {
    render(TransactionsPage, {
      data: { ...pending(), transactions: defer(Promise.reject(new Error('ledger tab missing'))) } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('ledger tab missing');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('searchbox', { name: 'Search transactions' })).toBeTruthy();
  });

  it('Given selected rows When Recategorise is clicked Then it stays busy until the sheet answers', async () => {
    render(TransactionsPage, {
      data: {
        ...pending(),
        transactions: settled(rows),
        options: settled({ accounts, categories, tags: [] }),
      } as never,
    });
    await flush();

    await fireEvent.click(screen.getByRole('checkbox', { name: 'Select Corner Market' }));
    await fireEvent.change(screen.getByRole('combobox', { name: 'New category' }), { target: { value: 'Transport' } });
    const apply = screen.getByRole('button', { name: 'Recategorise' });
    await fireEvent.click(apply);
    await flush();

    expect(calls.some((call) => call.name === 'updateTransaction')).toBe(true);
    expect(apply.getAttribute('aria-busy')).toBe('true');
    expect(apply.textContent).toContain('Updating…');
    expect(screen.getByRole('button', { name: 'Clear' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given a row to delete When Delete is confirmed Then the dialog button stays busy until the sheet answers', async () => {
    render(TransactionsPage, { data: { ...pending(), transactions: settled(rows) } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Delete Bus Card' }));
    await flush();
    const dialog = screen.getByRole('heading', { name: 'Delete transaction' }).closest('dialog') as HTMLElement;
    const remove = within(dialog).getByRole('button', { name: 'Delete' });
    await fireEvent.click(remove);
    await flush();

    expect(calls.find((call) => call.name === 'deleteTransaction')?.args).toEqual(['t2', 'etag-t2']);
    expect(remove.getAttribute('aria-busy')).toBe('true');
    expect(remove.textContent).toContain('Deleting…');
  });

  it('Given the ledger has loaded When amounts are masked Then payments stay hidden until toggled', async () => {
    render(TransactionsPage, { data: { ...pending(), transactions: settled(rows) } as never });
    await flush();

    expect(screen.getByText('2 transactions')).toBeTruthy();
    expect(document.body.textContent).not.toMatch(/734\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toMatch(/734\.000/);
  });
  describe('cards on a shared account', () => {
    // Invented set-up: two cards bill to one account until the second closes.
    const cardConfig = {
      programs: [{ name: 'Shared Programme', bank: 'Bank A', versions: [] }],
      banks: [{ name: 'Bank A', sharedAccount: 'Card A', referenceLimit: 0 }],
      cards: [
        { name: 'Emerald', program: 'Shared Programme', keyword: 'emerald', default: true },
        { name: 'Ruby', program: 'Shared Programme', keyword: 'ruby', closedFrom: '2026-03-01T00:00:00Z' },
      ],
      merchants: [],
    };
    const cardAccounts = [...accounts, { name: 'Card A', type: 'negative active use', currency: 'VND' } as Account];
    const onCard = (id: string, payee: string, date: string, tags: string[]): Transaction => ({
      ...tx(id, payee, 90_000),
      accountName: 'Card A',
      date,
      tags,
    });

    it('Given card rows When the ledger renders Then the card is named only while several were open and every tag shows', async () => {
      render(TransactionsPage, {
        data: {
          ...pending(),
          transactions: settled([
            onCard('c1', 'Noodle Bar', '2026-10-10', ['emerald', 'eating_out']),
            onCard('c2', 'Old Cafe', '2026-02-10', ['ruby', 'mcc:5812']),
          ]),
          options: settled({ accounts: cardAccounts, categories, tags: [] }),
          cardConfig: settled(cardConfig),
        } as never,
      });
      await flush();

      const recent = screen.getByText('Noodle Bar').closest('tr')!;
      expect(within(recent).getByText('eating_out')).toBeTruthy();
      expect(within(recent).getByText('emerald')).toBeTruthy();
      expect(within(recent).queryByText('Emerald')).toBeNull();
      const old = screen.getByText('Old Cafe').closest('tr')!;
      expect(within(old).getByText('Ruby · MCC 5812')).toBeTruthy();
      expect(within(old).getByText('ruby')).toBeTruthy();
      expect(within(old).getByText('mcc:5812')).toBeTruthy();
    });

    it('Given an account with several open cards When a payment is added Then the card is picked apart from the tags', async () => {
      render(TransactionsPage, {
        data: {
          ...pending(),
          transactions: settled([]),
          options: settled({ accounts: cardAccounts, categories, tags: ['eating_out', 'emerald', 'mcc:5812'] }),
          cardConfig: settled(cardConfig),
        } as never,
      });
      await flush();
      await fireEvent.click(screen.getByRole('button', { name: 'Add' }));
      await flush();

      await fireEvent.change(document.getElementById('tx-account')!, { target: { value: 'Card A' } });
      await fireEvent.input(document.getElementById('tx-date')!, { target: { value: '2026-02-10' } });
      await flush();
      const card = screen.getByLabelText<HTMLSelectElement>('Card');
      expect(card.value).toBe('emerald');
      await fireEvent.change(card, { target: { value: 'ruby' } });
      await fireEvent.input(screen.getByRole('textbox', { name: 'Add tag' }), { target: { value: 'a' } });
      await flush();
      expect(screen.queryByRole('button', { name: 'emerald' })).toBeNull();
      expect(screen.getByRole('button', { name: 'eating_out' })).toBeTruthy();
      expect(screen.getByTitle('ruby: the card that paid, not edited here')).toBeTruthy();
      expect(screen.queryByRole('button', { name: 'Remove ruby' })).toBeNull();

      await fireEvent.click(screen.getByRole('button', { name: 'eating_out' }));
      await fireEvent.input(document.getElementById('tx-payee')!, { target: { value: 'Old Cafe' } });
      await fireEvent.change(document.getElementById('tx-category')!, { target: { value: 'Groceries' } });
      await fireEvent.input(document.getElementById('tx-payment')!, { target: { value: '90000' } });
      await fireEvent.click(screen.getByRole('button', { name: 'Add transaction' }));
      await flush();

      const created = calls.find((call) => call.name === 'createTransaction');
      expect((created?.args[0] as { tags: string[] }).tags).toEqual(['ruby', 'eating_out']);
    });

    it('Given an account with one open card When a payment is added Then the card is filled without asking', async () => {
      render(TransactionsPage, {
        data: {
          ...pending(),
          transactions: settled([]),
          options: settled({ accounts: cardAccounts, categories, tags: [] }),
          cardConfig: settled(cardConfig),
        } as never,
      });
      await flush();
      await fireEvent.click(screen.getByRole('button', { name: 'Add' }));
      await flush();

      await fireEvent.change(document.getElementById('tx-account')!, { target: { value: 'Card A' } });
      await fireEvent.input(document.getElementById('tx-date')!, { target: { value: '2026-10-10' } });
      await flush();
      expect(screen.queryByLabelText('Card')).toBeNull();
      await fireEvent.input(document.getElementById('tx-payee')!, { target: { value: 'Noodle Bar' } });
      await fireEvent.change(document.getElementById('tx-category')!, { target: { value: 'Groceries' } });
      await fireEvent.input(document.getElementById('tx-payment')!, { target: { value: '90000' } });
      await fireEvent.click(screen.getByRole('button', { name: 'Add transaction' }));
      await flush();

      const created = calls.find((call) => call.name === 'createTransaction');
      expect((created?.args[0] as { tags: string[] }).tags).toEqual(['emerald']);
    });
  });
});
