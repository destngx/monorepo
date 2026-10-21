import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Bills, RecurringSuggestion } from '$lib/api/types';
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

import BillsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented figures.
const suggestion: RecurringSuggestion = {
  key: 'gym',
  name: 'Gym',
  kind: 'subscription',
  amount: 450_000,
  frequency: 'monthly',
  day: 5,
  category: 'Health',
  match: 'GYM',
  lastSeen: '2026-09',
  seen: 4,
  checked: 6,
  amountSpread: 0,
  daySpread: 1,
  confidence: 'high',
};
const bills: Bills = {
  asOf: '2026-10-17',
  tabExists: true,
  items: [],
  issues: [],
  suggestions: [suggestion],
  calendar: {
    from: '2026-10-01',
    to: '2026-11-16',
    occurrences: [],
    upcomingOut: 1_250_000,
    upcomingCount: 1,
    upcomingIn: 0,
    overdueCount: 0,
    overdueAmount: 0,
    paidOut: 0,
  },
  safeToSpend: {
    asOf: '2026-10-17',
    until: '2026-10-25',
    untilIncome: true,
    incomeName: 'Salary',
    days: 8,
    readyMoney: 9_870_000,
    readyAccounts: [
      { name: 'Bank A', balance: 12_870_000, reserved: 5_000_000, spendable: 7_870_000 },
      { name: 'Wallet', balance: 2_000_000, reserved: 0, spendable: 2_000_000 },
    ],
    excludedAccounts: [
      { name: 'Broker B', balance: 15_000_000, reason: 'not_ready' },
      { name: 'Exchange C', balance: 4_000_000, reason: 'crypto' },
    ],
    readiness: { notReadyAccounts: ['Broker B', 'Old Broker'], etag: 'etag-1' },
    reserved: 5_000_000,
    minimumBalance: {
      accounts: [
        { account: 'Bank A', minimum: 5_000_000, note: 'monthly reward' },
        { account: 'Old Bank', minimum: 1_000_000 },
      ],
      etag: 'etag-m',
    },
    minimumBalances: [
      {
        account: 'Bank A',
        minimum: 5_000_000,
        note: 'monthly reward',
        balance: 12_870_000,
        reserved: 5_000_000,
        month: '2026-10',
        monthEnd: '2026-10-31',
        daysInMonth: 31,
        daysCounted: 16,
        daysLeft: 15,
        sumSoFar: 96_000_000,
        averageSoFar: 6_000_000,
        projectedAverage: 9_324_194,
        requiredBalance: 3_933_334,
        extraNeeded: 0,
        status: 'on_track',
      },
    ],
    minimumRule: '',
    unknownMinimum: ['Old Bank'],
    billsDue: 1_250_000,
    cardPayments: 0,
    fixedCosts: 600_000,
    fixed: [
      {
        category: 'Rent',
        month: '2026-10',
        expected: 8_000_000,
        basis: 'budget',
        spent: 8_000_000,
        inBills: 0,
        amount: 0,
      },
      {
        category: 'Utilities',
        month: '2026-10',
        expected: 1_000_000,
        basis: 'budget',
        spent: 400_000,
        inBills: 0,
        amount: 600_000,
      },
    ],
    committed: 1_850_000,
    amount: 8_020_000,
    essentials: 800_000,
    essential: [{ category: 'Groceries', perDay: 100_000, basis: 'average', days: 8, amount: 800_000 }],
    safeToSpend: 7_220_000,
    shortBy: 0,
    tightBy: 0,
    perDay: 902_500,
    natureTagged: true,
    counted: [
      {
        name: 'Internet',
        kind: 'bill',
        source: 'recurring',
        expectedDate: '2026-10-10',
        expectedAmount: 250_000,
        status: 'overdue',
        daysUntil: -7,
        paidAmount: 0,
        remaining: 250_000,
      },
    ],
    explanation: '',
    readyRule: '',
    setAsideRule: '',
    unknownNotReady: ['Old Broker'],
    issues: ['Preferences: safeToSpend.notReadyAccounts names "Old Broker", which is not an account'],
  },
};

describe('Bills page', () => {
  afterEach(() => {
    mask.hide();
    calls.length = 0;
  });

  it('Given nothing has answered When the page renders Then the header and New item show with a skeleton per section', () => {
    render(BillsPage, { data: { engine: never(), bills: never(), idleCash: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Bills' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'New item' }).hasAttribute('disabled')).toBe(false);
    const loading = screen.getAllByRole('status').map((status) => status.textContent?.trim());
    expect(loading).toEqual(
      expect.arrayContaining([
        'Loading safe to spend…',
        'Loading idle cash…',
        'Loading next 30 days…',
        'Loading suggestions…',
        'Loading recurring items…',
      ]),
    );
  });

  it('Given the idle cash check fails When it settles Then it shows the error with Retry and the bills keep loading', async () => {
    render(BillsPage, {
      data: {
        engine: never(),
        bills: never(),
        idleCash: defer(Promise.reject(new Error('rates sheet missing'))),
      } as never,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('idle cash');
    expect(alert.textContent).toContain('rates sheet missing');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('Loading safe to spend…')).toBeTruthy();
  });

  it('Given a suggestion When Confirm is clicked Then only that button is busy until the sheet answers', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    const confirm = screen.getByRole('button', { name: 'Confirm' });
    await fireEvent.click(confirm);
    await flush();

    expect(calls.some((call) => call.name === 'confirmSuggestion')).toBe(true);
    expect(confirm.getAttribute('aria-busy')).toBe('true');
    expect(confirm.textContent).toContain('Confirming…');
    expect(screen.getByRole('button', { name: 'Dismiss' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given the bills have loaded When amounts are masked Then safe to spend stays hidden until toggled', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    expect(screen.getByText('Safe to spend until 25/10/2026')).toBeTruthy();
    expect(document.body.textContent).not.toMatch(/7\.220\.000|9\.870\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toMatch(/7\.220\.000/);
  });

  it('Given fixed and essential categories When the headline shows Then committed, essentials and free are broken down', async () => {
    mask.toggle();
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    const rows = [...screen.getByTestId('safe-breakdown').querySelectorAll('dt')].map((dt) => {
      const value = dt.nextElementSibling?.textContent ?? '';
      return `${(dt.textContent ?? '').trim()}: ${value.replace(/\s+/g, ' ').trim()}`;
    });
    expect(rows).toEqual([
      'Ready money, above minimum balances: 9.870.000 ₫',
      'Kept as minimum balances (not counted): 5.000.000 ₫',
      'Bills due before payday: − 1.250.000 ₫',
      'Card payments due before payday: − 0 ₫',
      'Fixed costs still to pay: − 600.000 ₫',
      'Left after committed: 8.020.000 ₫',
      'Essentials for 8 days: − 800.000 ₫',
      'Free to spend: 7.220.000 ₫',
    ]);
    const setAside = screen.getByTestId('set-aside').textContent?.replace(/\s+/g, ' ') ?? '';
    expect(setAside).toContain('Utilities, Oct 2026 budget 1.000.000 ₫, spent 400.000 ₫ 600.000 ₫');
    expect(setAside).toContain('Rent, Oct 2026 budget 8.000.000 ₫, spent 8.000.000 ₫ 0 ₫');
    expect(setAside).toContain('Groceries 100.000 ₫ a day (6-month average) for 8 days 800.000 ₫');
  });

  it('Given ready money When the breakdown shows Then each account has a switch and left-out accounts give the reason', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    const switches = screen.getAllByRole('switch');
    expect(switches.map((s) => s.getAttribute('aria-label'))).toEqual([
      'Bank A is ready to use',
      'Wallet is ready to use',
      'Broker B is not ready to use',
    ]);
    expect(switches.map((s) => s.getAttribute('aria-checked'))).toEqual(['true', 'true', 'false']);
    const breakdown = screen.getByTestId('ready-money').textContent ?? '';
    expect(breakdown).toContain('Left out: marked not ready to use (slow to withdraw)');
    expect(breakdown).toContain('Left out: crypto');
    expect(breakdown).toContain('Due before 25/10/2026');
    expect(breakdown).toContain('Internet, 10/10/2026, overdue');
    expect(breakdown).toContain('“Old Broker” is marked not ready but is not an account');
    expect(document.body.textContent).not.toMatch(/15\.000\.000|7\.870\.000/);
    const explanation = screen.getByTestId('safe-explanation').textContent ?? '';
    expect(explanation).toContain('anything due on payday itself is paid by that');
    expect(explanation.toLowerCase()).not.toContain('goal');
  });

  it('Given a ready account When its switch is clicked Then the change is sent with the ETag and only that switch is busy', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    const bank = screen.getByRole('switch', { name: 'Bank A is ready to use' });
    await fireEvent.click(bank);
    await flush();

    const sent = calls.find((call) => call.name === 'setAccountReady');
    expect(sent?.args).toEqual([{ account: 'Bank A', ready: false, etag: 'etag-1' }]);
    expect(bank.getAttribute('aria-busy')).toBe('true');
    expect(bank.textContent).toContain('Saving…');
    expect(screen.getByRole('switch', { name: 'Broker B is not ready to use' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given a minimum balance When the breakdown shows Then the reserve, the average and an on-track badge show, masked', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    expect(document.body.textContent).not.toMatch(/5\.000\.000|6\.000\.000/);
    mask.toggle();
    await flush();
    expect(screen.getByTestId('safe-reserved').textContent).toContain('5.000.000');
    expect(screen.getByTestId('reserved').textContent).toContain('5.000.000 ₫ kept as its minimum, 7.870.000 ₫ counts');
    const section = screen.getByTestId('minimum-balances');
    const rows = screen.getAllByTestId('minimum-row');
    expect(rows.map((row) => row.textContent?.split(/\s+/).filter(Boolean)[0])).toEqual(['Bank', 'Wallet', 'Broker']);
    expect(rows[0].textContent).toContain('Keeps 5.000.000 ₫, monthly reward');
    expect(rows[0].textContent).toContain('Average so far 6.000.000');
    expect(rows[0].querySelector('[data-status="good"]')?.textContent).toContain('On track');
    expect(section.textContent).not.toContain('Exchange C');
    expect(section.textContent).toContain('“Old Bank” has a minimum but is not an account');
    const explanation = (screen.getByTestId('safe-explanation').textContent ?? '').replace(/\s+/g, ' ');
    expect(explanation).toContain('counts only what is above that minimum: it is a hard reserve');
  });

  it('Given an account without a minimum When one is set Then the input starts empty and the change is sent with the ETag', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Set a minimum for Wallet' }));
    await flush();
    const amount = screen.getByLabelText<HTMLInputElement>('Minimum for Wallet in VND');
    expect(amount.value).toBe('');
    const save = screen.getByRole('button', { name: 'Save' });
    expect(save.hasAttribute('disabled')).toBe(true);
    await fireEvent.input(amount, { target: { value: '3.000.000' } });
    await fireEvent.input(screen.getByLabelText('Note for Wallet'), { target: { value: ' points ' } });
    await flush();
    await fireEvent.click(save);
    await flush();

    const sent = calls.find((call) => call.name === 'setAccountMinimum');
    expect(sent?.args).toEqual([{ account: 'Wallet', minimum: 3_000_000, note: 'points', etag: 'etag-m' }]);
    expect(save.getAttribute('aria-busy')).toBe('true');
    expect(save.textContent).toContain('Saving…');
    expect(screen.getByRole('button', { name: 'Change the minimum of Bank A' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given the minimum editor When it opens Then the amount has focus, Enter saves and Escape closes it', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    const toggle = screen.getByRole('button', { name: 'Set a minimum for Wallet' });
    await fireEvent.click(toggle);
    await flush();
    const amount = screen.getByLabelText('Minimum for Wallet in VND');
    expect(document.activeElement).toBe(amount);
    // Enter in a field clicks the form's submit button.
    expect(screen.getByRole('button', { name: 'Save' }).getAttribute('type')).toBe('submit');
    expect(amount.closest('form')).toBe(screen.getByRole('button', { name: 'Save' }).closest('form'));

    await fireEvent.keyDown(amount, { key: 'Escape' });
    await flush();
    expect(screen.queryByLabelText('Minimum for Wallet in VND')).toBeNull();
    expect(document.activeElement).toBe(toggle);
  });

  it('Given a minimum for a name that is no account When Remove is clicked Then its minimum is cleared', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Remove the minimum of Old Bank' }));
    await flush();
    const sent = calls.find((call) => call.name === 'setAccountMinimum');
    expect(sent?.args).toEqual([{ account: 'Old Bank', minimum: null, note: '', etag: 'etag-m' }]);
  });

  it('Given a marked name that is no account When Remove is clicked Then it is marked ready to clear it', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Remove Old Broker from the not-ready accounts' }));
    await flush();
    const sent = calls.find((call) => call.name === 'setAccountReady');
    expect(sent?.args).toEqual([{ account: 'Old Broker', ready: true, etag: 'etag-1' }]);
  });
});
