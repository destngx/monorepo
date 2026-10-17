import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '$lib/api/client';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { Recommendation, RecommendationList } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

// Every engine call stays pending unless a test settles it.
// A test can answer a call through `answers`.
const calls = vi.hoisted(() => [] as { name: string; args: unknown[] }[]);
const answers = vi.hoisted((): Record<string, (...args: unknown[]) => Promise<unknown>> => ({}));
vi.mock('$lib/api/endpoints', () => ({
  api: new Proxy(
    {},
    {
      get:
        (_, name) =>
        (...args: unknown[]) => {
          calls.push({ name: String(name), args });
          return answers[String(name)]?.(...args) ?? new Promise(() => undefined);
        },
    },
  ),
}));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import RecommendationsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

// Invented items and figures.
const item = (id: string, title: string, impactVnd: number | null = null): Recommendation => ({
  id,
  row: 2,
  created: '2026-10-01',
  source: 'budget-review',
  title,
  action: 'Do it this week',
  impactVnd,
  evidencePath: '/budget',
  evidence: 'Over its limit',
  status: 'open',
  page: '/budget',
  etag: `etag-${id}`,
});
const list = (items: Recommendation[]): RecommendationList => ({
  tabExists: true,
  items,
  summary: {
    open: items.length,
    snoozed: 0,
    done: 0,
    dismissed: 0,
    openImpactVnd: items.reduce((sum, entry) => sum + (entry.impactVnd ?? 0), 0),
    openWithoutImpact: items.filter((entry) => entry.impactVnd === null).length,
  },
  issues: [],
});
const data = (overrides: Record<string, unknown> = {}) =>
  ({ engine: never(), page: '', list: never(), refresh: never(), ...overrides }) as never;

describe('Recommendations page', () => {
  afterEach(() => {
    mask.hide();
    for (const name of Object.keys(answers)) delete answers[name];
  });

  it('Given nothing has answered When the page renders Then the header, the rule check status and a skeleton show', () => {
    render(RecommendationsPage, { data: data() });

    expect(screen.getByRole('heading', { level: 1, name: 'Recommendations' })).toBeTruthy();
    expect(screen.getByTestId('checking-rules').textContent).toContain('Checking the rules…');
    expect(screen.getByText('What should I do next?')).toBeTruthy();
    expect(screen.getByText('Loading recommendations…')).toBeTruthy();
  });

  it('Given the stored list is read When the rules are still running Then the list shows without waiting for them', async () => {
    render(RecommendationsPage, { data: data({ list: settled(list([item('rec-1', 'Cook at home')])) }) });
    await flush();

    expect(screen.getByText('Cook at home')).toBeTruthy();
    expect(screen.getByTestId('checking-rules')).toBeTruthy();
  });

  it('Given the stored list is shown When the rule refresh lands Then its list replaces it and the status goes', async () => {
    let finish: (value: Settled<RecommendationList>) => void = () => undefined;
    const refresh = new Promise<Settled<RecommendationList>>((resolve) => (finish = resolve));
    render(RecommendationsPage, { data: data({ list: settled(list([item('rec-1', 'Cook at home')])), refresh }) });
    await flush();

    finish({ ok: true, value: list([item('rec-1', 'Cook at home'), item('rec-2', 'Move idle cash')]) });
    await flush();

    expect(screen.getByText('Move idle cash')).toBeTruthy();
    expect(screen.queryByTestId('checking-rules')).toBeNull();
  });

  it('Given the refresh is refused for write access When it settles Then the stored list stays with a note', async () => {
    render(RecommendationsPage, {
      data: data({
        list: settled(list([item('rec-1', 'Cook at home')])),
        refresh: defer(Promise.reject(new ApiError(403, 'read-only connection'))),
      }),
    });
    await flush();

    expect(screen.getByText(/The engine checks did not run/)).toBeTruthy();
    expect(screen.getByText('Cook at home')).toBeTruthy();
    expect(screen.queryByRole('alert')).toBeNull();
  });

  it('Given the refresh fails otherwise When the list is shown Then an error strip offers Retry above it', async () => {
    render(RecommendationsPage, {
      data: data({
        list: settled(list([item('rec-1', 'Cook at home')])),
        refresh: defer(Promise.reject(new Error('rules crashed'))),
      }),
    });
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Could not refresh the engine checks: rules crashed');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByText('Cook at home')).toBeTruthy();
  });

  it('Given the read and the refresh both fail When they settle Then the list shows its error with Retry', async () => {
    render(RecommendationsPage, {
      data: data({
        list: defer(Promise.reject(new Error('tab unreadable'))),
        refresh: defer(Promise.reject(new Error('rules crashed'))),
      }),
    });
    await flush();

    const alerts = screen.getAllByRole('alert').map((alert) => alert.textContent ?? '');
    expect(
      alerts.some((text) => text.includes('Could not load recommendations') && text.includes('tab unreadable')),
    ).toBe(true);
    expect(screen.getAllByRole('button', { name: /Retry/ }).length).toBe(2);
  });

  it('Given an open item When Done is pressed Then that button spins and the others wait', async () => {
    render(RecommendationsPage, {
      data: data({ list: settled(list([item('rec-1', 'Cook at home'), item('rec-2', 'Move idle cash')])) }),
    });
    await flush();
    calls.length = 0;

    const [done, otherDone] = screen.getAllByRole('button', { name: 'Done' });
    await fireEvent.click(done);
    await flush();

    expect(calls).toEqual([{ name: 'setRecommendationStatus', args: ['rec-1', 'etag-rec-1', { status: 'done' }] }]);
    expect(done.getAttribute('aria-busy')).toBe('true');
    expect(otherDone.hasAttribute('disabled')).toBe(true);
  });

  it('Given items with an impact When masked Then the amounts stay hidden until toggled', async () => {
    render(RecommendationsPage, { data: data({ list: settled(list([item('rec-1', 'Cook at home', 1_200_000)])) }) });
    await flush();

    expect(document.body.textContent).toContain(MASK);
    expect(document.body.textContent).not.toMatch(/1\.200\.000/);
    await fireEvent.click(screen.getAllByRole('button', { name: 'Hidden amount, click to reveal' })[0]);
    await flush();
    expect(document.body.textContent).toContain('1.200.000');
  });

  it('Given an item was marked done When an older rule refresh lands afterwards Then the list is re-read, not reverted', async () => {
    const open = item('rec-1', 'Cook at home');
    const doneItem: Recommendation = { ...open, status: 'done', etag: 'etag-2' };
    let finish: (value: Settled<RecommendationList>) => void = () => undefined;
    const refresh = new Promise<Settled<RecommendationList>>((resolve) => (finish = resolve));
    answers.setRecommendationStatus = () => Promise.resolve(doneItem);
    answers.recommendations = () => Promise.resolve({ ...list([doneItem]), summary: { ...list([]).summary, done: 1 } });
    render(RecommendationsPage, { data: data({ list: settled(list([open])), refresh }) });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: 'Done' }));
    await flush();
    expect(screen.getByRole('button', { name: 'Reopen' })).toBeTruthy();

    calls.length = 0;
    finish({ ok: true, value: list([open]) });
    await flush();

    expect(calls.map((call) => call.name)).toEqual(['recommendations']);
    expect(screen.getByRole('button', { name: 'Reopen' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: 'Done' })).toBeNull();
  });
});
