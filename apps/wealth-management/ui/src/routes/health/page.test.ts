import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { FinancialHealth } from '$lib/api/types';
import { MASK } from '$lib/format';
import { netWorthHistory } from '$lib/networth/history.testdata';
import { mask } from '$lib/state/mask.svelte';

// The recommendations panel loads on its own and stays pending.
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy({}, { get: () => () => new Promise(() => undefined) }),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import HealthPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented figures.
const health: FinancialHealth = {
  score: 68,
  uncappedScore: 68,
  band: 'yellow',
  caps: [],
  components: [
    {
      key: 'emergency',
      label: 'Emergency fund',
      value: 3.2,
      target: 6,
      unit: 'months',
      score: 13,
      maxScore: 25,
      formula: 'liquid / monthly expense',
      detail: 'Liquid 16.000.000 ₫ over 5.000.000 ₫ a month',
    },
  ],
  recommendations: [{ component: 'emergency', title: 'Build the buffer', action: 'Set aside 1.000.000 ₫ a month' }],
  liquidAssets: 16_000_000,
  avgMonthlyIncome: 9_000_000,
  avgMonthlyExpense: 5_000_000,
  netWorth: 40_000_000,
  freedomMonths: 3.2,
  allocation: {
    lines: [],
    total: 0,
    targetedTotal: 0,
    targets: null,
    driftThreshold: 5,
    rebalance: false,
    unvalued: 0,
    issues: [],
    generatedAt: '2026-10-15T08:00:00Z',
  },
  lookbackMonths: 6,
  insufficientIncome: false,
};

describe('Health page', () => {
  afterEach(() => mask.hide());

  it('Given a cap holds the score When it renders Then the reason and the uncapped sum show', async () => {
    const held: FinancialHealth = {
      ...health,
      score: 49,
      uncappedScore: 81,
      band: 'red',
      caps: [{ key: 'noBuffer', limit: 49, reason: 'Liquid cash covers 0.8 months of expense, under 1 month.' }],
    };
    render(HealthPage, { data: { engine: never(), health: settled(held), netWorthHistory: never() } as never });
    await flush();

    const caps = screen.getByTestId('health-caps').textContent?.replace(/\s+/g, ' ') ?? '';
    expect(caps).toContain('Held at 49: the pillars add up to 81');
    expect(caps).toContain('under 1 month. At most 49.');
  });

  it('Given nothing has answered When the page renders Then the header shows with the score and history skeletons', () => {
    render(HealthPage, { data: { engine: never(), health: never(), netWorthHistory: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Financial Health' })).toBeTruthy();
    expect(screen.getByText('Loading health score…')).toBeTruthy();
    expect(screen.getByText('Loading net worth history…')).toBeTruthy();
  });

  it('Given the score loads first When the history is still pending Then the score shows and the history keeps loading', async () => {
    render(HealthPage, { data: { engine: never(), health: settled(health), netWorthHistory: never() } as never });
    await flush();

    expect(screen.getByText('Computed from the last 6 months of transactions')).toBeTruthy();
    expect(screen.getByText('68')).toBeTruthy();
    expect(screen.getByText('Loading net worth history…')).toBeTruthy();
  });

  it('Given the history fails When the score has loaded Then only the history shows its error and Retry', async () => {
    render(HealthPage, {
      data: {
        engine: never(),
        health: settled(health),
        netWorthHistory: defer(Promise.reject(new Error('snapshots tab unreadable'))),
      } as never,
    });
    await flush();

    const alerts = screen.getAllByRole('alert');
    expect(alerts).toHaveLength(1);
    expect(alerts[0].textContent).toContain('Could not load the net worth history');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('68')).toBeTruthy();
  });

  it('Given the score fails When the history has loaded Then one error with Retry shows and the history renders', async () => {
    render(HealthPage, {
      data: {
        engine: never(),
        health: defer(Promise.reject(new Error('engine busy'))),
        netWorthHistory: settled(netWorthHistory()),
      } as never,
    });
    await flush();

    expect(screen.getAllByRole('alert')).toHaveLength(1);
    expect(screen.getByRole('alert').textContent).toContain('Could not load the health score');
    expect(screen.getByTestId('networth-headline')).toBeTruthy();
  });

  it('Given the score has loaded When masked Then amounts in stats, the math and the orders stay hidden until toggled', async () => {
    render(HealthPage, { data: { engine: never(), health: settled(health), netWorthHistory: never() } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Show the math' }));
    expect(screen.getByRole('button', { name: 'Hide the math' }).getAttribute('aria-expanded')).toBe('true');
    expect(document.body.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/16\.000\.000|1\.000\.000/);

    mask.toggle();
    await flush();
    expect(document.body.textContent).toContain('16.000.000');
    expect(document.body.textContent).toContain('Set aside 1.000.000 ₫ a month');
  });
});
