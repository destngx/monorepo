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
      { name: 'Bank A', balance: 7_870_000 },
      { name: 'Wallet', balance: 2_000_000 },
    ],
    excludedAccounts: [
      { name: 'Broker B', balance: 15_000_000, reason: 'not_ready' },
      { name: 'Exchange C', balance: 4_000_000, reason: 'crypto' },
    ],
    readiness: { notReadyAccounts: ['Broker B', 'Old Broker'], etag: 'etag-1' },
    billsDue: 1_250_000,
    cardPayments: 0,
    amount: 8_620_000,
    safeToSpend: 8_620_000,
    shortBy: 0,
    perDay: 1_077_500,
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
    expect(document.body.textContent).not.toMatch(/8\.620\.000|9\.870\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toMatch(/8\.620\.000/);
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

  it('Given a marked name that is no account When Remove is clicked Then it is marked ready to clear it', async () => {
    render(BillsPage, { data: { engine: never(), bills: settled(bills), idleCash: never() } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Remove Old Broker from the not-ready accounts' }));
    await flush();
    const sent = calls.find((call) => call.name === 'setAccountReady');
    expect(sent?.args).toEqual([{ account: 'Old Broker', ready: true, etag: 'etag-1' }]);
  });
});
