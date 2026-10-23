import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { MoneyLent } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import LentPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented person and figures.
const lent: MoneyLent = {
  asOf: '2026-10-15',
  owedToYou: 3_400_000,
  youOwe: 0,
  peopleOwing: 1,
  peopleYouOwe: 0,
  oldestDaysOutstanding: 40,
  people: [
    {
      name: 'Test Person',
      lent: 3_400_000,
      repaid: 0,
      outstanding: 3_400_000,
      lastActivity: '2026-09-05',
      openSince: '2026-09-05',
      daysOutstanding: 40,
      repayBy: '2026-10-13',
      daysUntilRepay: -2,
      repayByAmount: 3_400_000,
      loans: [
        {
          transactionId: 'row-2',
          date: '2026-09-05',
          reference: null,
          amount: 3_400_000,
          repaid: 0,
          outstanding: 3_400_000,
          daysOutstanding: 40,
          repayBy: '2026-10-13',
          daysUntilRepay: -2,
          repayments: [],
        },
      ],
      transactions: [],
    },
  ],
};

describe('Money lent page', () => {
  afterEach(() => mask.hide());

  it('Given the ledger is still loading When the page renders Then the header shows with skeletons', () => {
    render(LentPage, { data: { engine: never(), lent: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Money lent' })).toBeTruthy();
    expect(screen.getByText('Loading what is owed to you…')).toBeTruthy();
    expect(screen.getByText('Loading open balances…')).toBeTruthy();
  });

  it('Given the ledger fails When it settles Then its error and Retry show in its place', async () => {
    render(LentPage, {
      data: { engine: never(), lent: defer(Promise.reject(new Error('lending tab unreadable'))) } as never,
    });
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Could not load money lent');
    expect(screen.getByRole('alert').textContent).toContain('lending tab unreadable');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('heading', { level: 1, name: 'Money lent' })).toBeTruthy();
  });

  it('Given the ledger has loaded When masked Then amounts and names stay hidden until toggled', async () => {
    render(LentPage, { data: { engine: never(), lent: settled(lent) } as never });
    await flush();

    expect(document.body.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/3\.400\.000|Test Person/);

    await fireEvent.click(screen.getAllByRole('button', { name: 'Hidden amount, click to reveal' })[0]);
    await flush();
    expect(document.body.textContent).toContain('3.400.000');
    expect(document.body.textContent).toContain('Test Person');
  });

  it('Given a repayment date in a memo When it has passed Then the row shows the date as overdue', async () => {
    render(LentPage, { data: { engine: never(), lent: settled(lent) } as never });
    await flush();

    expect(screen.getByRole('columnheader', { name: 'Repay by' })).toBeTruthy();
    expect(screen.getByText('2 days overdue')).toBeTruthy();
    expect(screen.queryByText(/more loan/)).toBeNull();
  });

  it('Given several loans with their own dates When the row renders Then it shows the soonest and points to the rest', async () => {
    const [person] = lent.people;
    const later = { ...person.loans[0], transactionId: 'row-3', date: '2026-10-01', amount: 1_000_000 };
    const several: MoneyLent = {
      ...lent,
      people: [
        {
          ...person,
          repayByAmount: 2_400_000,
          loans: [
            { ...person.loans[0], amount: 2_400_000, outstanding: 2_400_000 },
            { ...later, outstanding: 1_000_000, repayBy: '2026-11-30', daysUntilRepay: 46 },
          ],
        },
      ],
    };
    mask.toggle();
    render(LentPage, { data: { engine: never(), lent: settled(several) } as never });
    await flush();

    expect(screen.getByText('2 days overdue')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: '1 more loan, see details' }));
    const loans = screen.getByRole('table', { name: 'Loans' });
    expect(loans.textContent).toContain('30/11/2026');
    expect(loans.textContent).toContain('Due in 46 days');
    expect(screen.getByRole('button', { name: 'Hide transactions' }).getAttribute('aria-expanded')).toBe('true');
  });

  it('Given a person row When the history toggle is pressed Then it says what it does and expands', async () => {
    render(LentPage, { data: { engine: never(), lent: settled(lent) } as never });
    await flush();

    const toggle = screen.getByRole('button', { name: 'Show transactions' });
    expect(toggle.getAttribute('title')).toBe('Show transactions');
    await fireEvent.click(toggle);
    expect(screen.getByRole('button', { name: 'Hide transactions' }).getAttribute('aria-expanded')).toBe('true');
  });

  it('Given a repayment split over two loans When the history opens Then each loan lists it and the row names both loans', async () => {
    const [person] = lent.people;
    const repayment = { id: 'row-4', date: '2026-10-10', accountName: 'Bank', deposit: 1_500_000, memo: null };
    const paid: MoneyLent = {
      ...lent,
      people: [
        {
          ...person,
          loans: [
            {
              ...person.loans[0],
              amount: 1_000_000,
              repaid: 1_000_000,
              outstanding: 0,
              repayBy: null,
              daysUntilRepay: null,
              repayments: [{ transactionId: 'row-4', date: '2026-10-10', amount: 1_000_000 }],
            },
            {
              ...person.loans[0],
              transactionId: 'row-3',
              reference: 'L2',
              repaid: 500_000,
              outstanding: 2_900_000,
              repayments: [{ transactionId: 'row-4', date: '2026-10-10', amount: 500_000 }],
            },
          ],
          transactions: [repayment as never],
        },
      ],
    };
    mask.toggle();
    render(LentPage, { data: { engine: never(), lent: settled(paid) } as never });
    await flush();
    await fireEvent.click(screen.getByRole('button', { name: 'Show transactions' }));

    const loans = screen.getByRole('table', { name: 'Loans' });
    expect(loans.textContent).toMatch(/#1.*10\/10\/2026\s*1\.000\.000/s);
    expect(loans.textContent).toMatch(/L2.*10\/10\/2026\s*500\.000/s);
    const row = within(screen.getByRole('table', { name: 'Transactions' })).getAllByRole('row')[1];
    expect(row.lastElementChild?.textContent).toMatch(/#1\s*1\.000\.000.*L2\s*500\.000/s);
  });
});
