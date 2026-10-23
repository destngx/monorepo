import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { GoalPlan, GoalProgress } from '$lib/api/types';
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

import GoalsPage from './+page.svelte';
import { pendingFills } from '$lib/fill/pending.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented goals and figures.
const goal = (id: string, name: string, priority: number): GoalProgress => ({
  id,
  row: priority + 1,
  name,
  target: 50_000_000,
  priority,
  startingAmount: 0,
  status: 'active',
  etag: `etag-${id}`,
  useSlowMoney: false,
  emergencyFund: false,
  targetAmount: 50_000_000,
  targetComputed: false,
  funded: 12_300_000,
  fromPool: 12_300_000,
  fromSlow: 0,
  remaining: 37_700_000,
  percent: 24.6,
  monthsLeft: null,
  requiredPerMonth: null,
  projectedFinish: null,
  monthsToReach: null,
  yearsToReach: null,
  longHorizon: false,
  pace: 'no_deadline',
});
const plan = (overrides: Partial<GoalPlan> = {}): GoalPlan => ({
  asOf: '2026-10-15',
  setupNeeded: false,
  legacyGoals: 0,
  settings: { emergencyMonths: 6, spendingBufferMonths: 1 },
  pool: {
    assets: 30_000_000,
    cardDebt: 0,
    spendingBuffer: 5_000_000,
    bufferMonths: 1,
    averageMonthlyExpense: 5_000_000,
    total: 25_000_000,
    slow: 0,
    crypto: 0,
    archived: 0,
    moneyLent: 0,
    negativeBalances: 0,
    accounts: [],
  },
  needs: { perMonth: 4_000_000, months: 6, fromTotalExpense: false },
  monthlySavings: { perMonth: 2_000_000, months: 6 },
  goals: [goal('goal-1', 'Car', 1), goal('goal-2', 'Trip', 2)],
  unallocated: 700_000,
  unallocatedSlow: 0,
  check: {
    requiredPerMonth: 0,
    monthlySavings: 2_000_000,
    gap: 0,
    verdict: 'no_deadlines',
    datedGoals: 0,
    tradeOffs: [],
  },
  issues: [],
  ...overrides,
});

describe('Goals page', () => {
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

  it('Given the plan is still loading When the page renders Then the header and a waiting New goal show at once', () => {
    render(GoalsPage, { data: { engine: never(), plan: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Goals' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'New goal' }).hasAttribute('disabled')).toBe(true);
    expect(screen.getByText('Am I on track?')).toBeTruthy();
    expect(screen.getByText('Loading savings pool…')).toBeTruthy();
  });

  it('Given the plan fails When it settles Then its error and Retry show in its place', async () => {
    render(GoalsPage, {
      data: { engine: never(), plan: defer(Promise.reject(new Error('goal plan tab unreadable'))) } as never,
    });
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Could not load the goal plan');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('heading', { level: 1, name: 'Goals' })).toBeTruthy();
  });

  it('Given the plan has loaded When masked Then funded amounts stay hidden until toggled', async () => {
    render(GoalsPage, { data: { engine: never(), plan: settled(plan()) } as never });
    await flush();

    expect(screen.getByRole('button', { name: 'New goal' }).hasAttribute('disabled')).toBe(false);
    expect(document.body.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/12\.300\.000/);

    await fireEvent.click(screen.getAllByRole('button', { name: 'Hidden amount, click to reveal' })[0]);
    await flush();
    expect(document.body.textContent).toContain('12.300.000');
  });

  it('Given two active goals When the second moves up Then only its arrow spins and the other arrows wait', async () => {
    render(GoalsPage, { data: { engine: never(), plan: settled(plan()) } as never });
    await flush();
    calls.length = 0;

    const up = screen.getByRole('button', { name: 'Move Trip up' });
    expect(up.getAttribute('title')).toBe('Move Trip up');
    await fireEvent.click(up);
    await flush();

    expect(calls).toEqual([{ name: 'reorderGoals', args: [['goal-2', 'goal-1']] }]);
    expect(up.getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole('button', { name: 'Move Car down' }).hasAttribute('disabled')).toBe(true);
  });

  it('Given a plan not set up When Set up is pressed Then the button stays busy until the engine answers', async () => {
    render(GoalsPage, { data: { engine: never(), plan: settled(plan({ setupNeeded: true, goals: [] })) } as never });
    await flush();
    expect(screen.queryByRole('button', { name: 'New goal' })).toBeNull();

    const setup = screen.getByRole('button', { name: 'Set up your goal plan' });
    await fireEvent.click(setup);
    await flush();
    expect(setup.getAttribute('aria-busy')).toBe('true');
    expect(setup.textContent).toContain('Setting up…');
  });

  it('Given the new goal form When it is submitted blank Then the name shows as invalid', async () => {
    render(GoalsPage, { data: { engine: never(), plan: settled(plan()) } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'New goal' }));
    await fireEvent.submit(document.getElementById('goal-form') as HTMLFormElement);
    await flush();

    expect(document.getElementById('goal-name')?.getAttribute('aria-invalid')).toBe('true');
    expect(document.getElementById('goal-target')?.getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByText('Enter a name for the goal.')).toBeTruthy();
  });

  it('Given a goal that uses slow money When the plan shows Then its card and the pool show the slow money', async () => {
    const house = { ...goal('goal-1', 'House', 1), useSlowMoney: true, fromSlow: 8_000_000 };
    const pool = { ...plan().pool, slow: 9_000_000 };
    render(GoalsPage, {
      data: { engine: never(), plan: settled(plan({ goals: [house], pool, unallocatedSlow: 1_000_000 })) } as never,
    });
    await flush();
    mask.toggle();
    await flush();

    expect(screen.getByText('Uses slow money')).toBeTruthy();
    expect(screen.getByText('From slow money')).toBeTruthy();
    expect(screen.getByText('Slow money left')).toBeTruthy();
    expect(document.body.textContent).toContain('9.000.000');
  });

  it('Given a wallet below 0 When the pool shows Then it is flagged apart from card debt', async () => {
    const pool = {
      ...plan().pool,
      negativeBalances: 1_000_000,
      accounts: [{ name: 'Wallet', balance: -1_000_000, role: 'negative' as const }],
    };
    render(GoalsPage, { data: { engine: never(), plan: settled(plan({ pool })) } as never });
    await flush();

    expect(screen.getByText('Negative balances')).toBeTruthy();
    expect(screen.getByText('Below 0, check its transactions')).toBeTruthy();
  });

  it('Given a plan When it shows Then the money left and monthly needs sit in the headline', async () => {
    const pool = { ...plan().pool, slow: 9_000_000 };
    render(GoalsPage, { data: { engine: never(), plan: settled(plan({ pool })) } as never });
    await flush();

    const headline = document.querySelector('[aria-labelledby="goals-verdict"]');
    expect(headline?.textContent).toContain('Left after goals');
    expect(headline?.textContent).toContain('Slow money left');
    expect(headline?.textContent).toContain('Monthly needs');
  });

  it('Given the new goal form When Can use slow money is ticked Then the goal is created with it', async () => {
    calls.length = 0;
    render(GoalsPage, { data: { engine: never(), plan: settled(plan()) } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'New goal' }));
    await fireEvent.input(document.getElementById('goal-name') as HTMLInputElement, { target: { value: 'House' } });
    await fireEvent.input(document.getElementById('goal-target') as HTMLInputElement, {
      target: { value: '100000000' },
    });
    await fireEvent.click(screen.getByLabelText('Can use slow money'));
    await fireEvent.submit(document.getElementById('goal-form') as HTMLFormElement);
    await flush();

    const create = calls.find((call) => call.name === 'createGoal');
    expect(create?.args[0]).toMatchObject({ name: 'House', target: 100_000_000, useSlowMoney: true });
  });
});

describe('Goals page: a chat fill (WM-FILL-7)', () => {
  beforeAll(() => {
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

  it('Given the chat filled a goal When the page opens Then New goal opens with the values in it', async () => {
    pendingFills.set({
      form: 'goal',
      title: 'Goal',
      page: '/accounts/goals',
      text: 'save 50M for a trip',
      result: { form: 'goal', values: { name: 'Trip', target: 50_000_000 }, filled: ['name', 'target'], notes: [] },
    });
    render(GoalsPage, { data: { engine: never(), plan: settled(plan()) } as never });
    await flush();

    expect(document.querySelector('dialog')?.textContent).toContain('New goal');
    await vi.waitFor(() => expect(screen.getByDisplayValue('Trip')).toBeTruthy());
    expect(screen.getByDisplayValue('50000000')).toBeTruthy();
  });
});
