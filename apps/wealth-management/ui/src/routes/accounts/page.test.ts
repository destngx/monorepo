import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Account, CardBankReport, DashboardSummary } from '$lib/api/types';
import { MASK } from '$lib/format';
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
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import AccountsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const pendingData = () => ({
  engine: never(),
  accounts: never<Account[]>(),
  histories: never(),
  summary: never<DashboardSummary>(),
  allocation: never(),
  cards: never<CardBankReport[]>(),
});

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented figures and names.
const account = (name: string, balance: number): Account => ({
  name,
  dueDate: null,
  creditLimit: null,
  clearedBalance: balance,
  balance,
  type: 'active use',
  currency: 'VND',
  note: null,
});
const summary = {
  balanceSheet: { assets: 9_100_000, liabilities: 1_300_000, netWorth: 7_800_000, receivables: 0, payables: 0 },
} as unknown as DashboardSummary;

describe('Accounts page', () => {
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
  afterEach(() => mask.hide());

  it('Given no call has answered When the page renders Then the header, its actions and a skeleton per section show', () => {
    render(AccountsPage, { data: pendingData() as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Accounts' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'New account' })).toBeTruthy();
    expect(screen.getByRole('link', { name: /Money lent/ })).toBeTruthy();
    const loading = screen.getAllByRole('status').map((status) => status.textContent?.trim());
    expect(loading).toEqual(
      expect.arrayContaining([
        'Loading net worth…',
        'Loading asset allocation…',
        'Loading liquidity…',
        'Loading balance history…',
        'Loading accounts…',
      ]),
    );
  });

  it('Given the allocation fails When it settles Then its card shows the error with Retry and the rest keep loading', async () => {
    render(AccountsPage, {
      data: { ...pendingData(), allocation: defer(Promise.reject(new Error('market data down'))) } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('Could not load the asset allocation');
    expect(alert.textContent).toContain('market data down');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('Loading accounts…')).toBeTruthy();
  });

  it('Given the balance sheet has loaded When amounts are masked Then they stay hidden until toggled', async () => {
    render(AccountsPage, { data: { ...pendingData(), summary: settled(summary) } as never });
    await flush();

    expect(document.body.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/7\.800\.000/);

    await fireEvent.click(screen.getAllByRole('button', { name: 'Hidden amount, click to reveal' })[0]);
    await flush();
    expect(document.body.textContent).toContain('7.800.000');
  });

  it('Given the accounts have loaded When a delete is confirmed Then the Delete button stays busy until the engine answers', async () => {
    render(AccountsPage, {
      data: { ...pendingData(), accounts: settled([account('Wallet', 500_000)]) } as never,
    });
    await flush();
    expect(screen.getByText('1 account')).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Delete Wallet' }));
    await flush();
    calls.length = 0;
    const confirm = screen.getByRole('button', { name: 'Delete' });
    await fireEvent.click(confirm);
    await flush();

    expect(calls.map((call) => call.name)).toEqual(['deleteAccount']);
    expect(confirm.getAttribute('aria-busy')).toBe('true');
    expect(confirm.textContent).toContain('Deleting…');
  });

  it('Given a year of balance history When an account row shows Then its sparkline fills the column and spans 12 months', async () => {
    const year = Array.from({ length: 366 }, (_, day) => ({
      date: `2025-10-${String((day % 28) + 1).padStart(2, '0')}`,
      value: 100_000 + day * 1_000,
    }));
    render(AccountsPage, {
      data: {
        ...pendingData(),
        accounts: settled([account('Wallet', 466_000)]),
        histories: settled({ Wallet: year }),
      } as never,
    });
    await flush();

    const spark = screen.getByRole('img', { name: 'Wallet balance, last 12 months' });
    expect(spark.getAttribute('width')).toBe('100%');
    expect(spark.getAttribute('preserveAspectRatio')).toBe('none');
    const line = spark.querySelectorAll('path')[1];
    expect(line.getAttribute('d')?.match(/[ML]/g)).toHaveLength(366);
  });

  describe('credit card rows', () => {
    const card = (name: string, balance: number, extra: Partial<Account> = {}): Account => ({
      ...account(name, balance),
      type: 'negative active use',
      ...extra,
    });
    const accounts = [
      card('Card Acme', -300_000, { note: 'Còn 9.000.000' }),
      card('Card Acme Extra', -200_000),
      card('Card Noted', -100_000, { note: 'Còn 700.000' }),
      card('Card Bare', -400_000),
    ];
    const reports = [
      {
        limit: 5_000_000,
        cards: [
          { accountName: 'Card Acme', keyword: 'platinum' },
          { accountName: 'Card Acme Extra', keyword: 'gold' },
        ],
      },
    ] as unknown as CardBankReport[];
    const row = (name: string) =>
      (screen.getByText(name, { selector: 'p' }).closest('li')?.textContent ?? '').replace(/\s+/g, ' ');

    it('Given a bank limit shared by two card accounts When the cards load Then each row shows the limit less what both owe', async () => {
      mask.toggle();
      render(AccountsPage, {
        data: { ...pendingData(), accounts: settled(accounts), cards: settled(reports) } as never,
      });
      await flush();

      expect(row('Card Acme')).toContain('Available 4.500.000 ₫ of 5.000.000 ₫');
      expect(row('Card Acme')).toContain('Limit shared with Card Acme Extra');
      expect(row('Card Acme')).not.toContain('9.000.000');
      expect(row('Card Acme Extra')).toContain('Available 4.500.000 ₫ of 5.000.000 ₫');
    });

    it('Given a card with no limit When the cards load Then it shows the note amount, or no Available instead of the debt', async () => {
      mask.toggle();
      render(AccountsPage, {
        data: { ...pendingData(), accounts: settled(accounts), cards: settled(reports) } as never,
      });
      await flush();

      expect(row('Card Noted')).toContain('Available 700.000 ₫, from the note');
      expect(row('Card Bare')).not.toContain('Available');
    });

    it('Given the card set-up is still loading When the accounts show Then no card row shows an Available amount yet', async () => {
      mask.toggle();
      render(AccountsPage, { data: { ...pendingData(), accounts: settled(accounts) } as never });
      await flush();

      expect(row('Card Acme')).not.toContain('Available');
      expect(row('Card Noted')).not.toContain('Available');
    });
  });
});
