import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled } from '../load/load';

const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);

import Section from './Deferred.fixture.svelte';

type Value = { total: number };

/** A deferred promise the test settles when it wants. */
function pending() {
  let resolve!: (result: Settled<Value>) => void;
  const promise = new Promise<Settled<Value>>((done) => (resolve = done));
  return { promise, resolve };
}

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Deferred', () => {
  it('Given data still loading When rendered Then a busy skeleton with screen-reader text shows', () => {
    const { container } = render(Section, { data: pending().promise });

    expect(screen.getByRole('status').textContent).toContain('Loading net worth…');
    expect(container.querySelector('[data-state]')?.getAttribute('aria-busy')).toBe('true');
    expect(screen.queryByTestId('value')).toBeNull();
  });

  it('Given data that loads When it resolves Then the content replaces the skeleton', async () => {
    const data = pending();
    const { container } = render(Section, { data: data.promise });

    data.resolve({ ok: true, value: { total: 5 } });
    await flush();

    expect(screen.getByTestId('value').textContent).toBe('Total 5');
    expect(screen.queryByRole('status')).toBeNull();
    expect(container.querySelector('[data-state]')?.getAttribute('aria-busy')).toBe('false');
  });

  it('Given a request that fails When it settles Then the error shows in place with a Retry', async () => {
    const retry = vi.fn(() => Promise.resolve());
    render(Section, {
      data: defer(Promise.reject(Object.assign(new Error('engine broke'), { status: 500, code: 'INTERNAL' }))),
      retry,
    });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('Could not load net worth');
    expect(alert.textContent).toContain('engine broke');
    await fireEvent.click(screen.getByRole('button', { name: /Retry/ }));
    expect(retry).toHaveBeenCalledOnce();
  });

  it('Given no retry is passed When Retry is clicked Then the page load re-runs', async () => {
    navigation.invalidateAll.mockClear();
    render(Section, { data: defer(Promise.reject(new Error('timeout'))) });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: /Retry/ }));
    expect(navigation.invalidateAll).toHaveBeenCalledOnce();
  });

  it('Given a shown value When a refresh is in flight Then the value stays with a Refreshing badge, no skeleton', async () => {
    const { rerender, container } = render(Section, { data: settled({ total: 1 }) });
    await flush();

    const next = pending();
    await rerender({ data: next.promise });
    await flush();
    expect(screen.getByTestId('value').textContent).toBe('Total 1');
    expect(container.querySelector('[data-refreshing]')).toBeTruthy();
    expect(container.querySelector('[data-skeleton]')).toBeNull();

    next.resolve({ ok: true, value: { total: 2 } });
    await flush();
    expect(screen.getByTestId('value').textContent).toBe('Total 2');
    expect(container.querySelector('[data-refreshing]')).toBeNull();
  });

  it('Given a shown value When the refresh fails Then the value stays under an error strip', async () => {
    const { rerender } = render(Section, { data: settled({ total: 1 }) });
    await flush();

    await rerender({ data: defer(Promise.reject(new Error('sheet timeout'))) });
    await flush();
    expect(screen.getByTestId('value').textContent).toBe('Total 1');
    expect(screen.getByRole('alert').textContent).toContain('Could not refresh net worth: sheet timeout');
  });

  it('Given a shown value When the scope changes Then the skeleton returns instead of stale data', async () => {
    const { rerender } = render(Section, { data: settled({ total: 1 }), scope: '2026-09' });
    await flush();

    await rerender({ data: pending().promise, scope: '2026-10' });
    await flush();
    expect(screen.queryByTestId('value')).toBeNull();
    expect(screen.getByRole('status').textContent).toContain('Loading net worth…');
  });

  it('Given an older request that resolves late When a newer one already settled Then the newer value wins', async () => {
    const older = pending();
    const { rerender } = render(Section, { data: older.promise });
    await rerender({ data: settled({ total: 2 }) });
    await flush();

    older.resolve({ ok: true, value: { total: 1 } });
    await flush();
    expect(screen.getByTestId('value').textContent).toBe('Total 2');
  });

  it('Given a request that cannot reach the engine When it settles Then the error says what to do', async () => {
    render(Section, { data: defer(Promise.reject(new TypeError('Failed to fetch'))) });
    await flush();

    const alert = screen.getByRole('alert');
    expect(alert.textContent).toContain('Engine unreachable: Failed to fetch');
    expect(alert.textContent).toContain('Check that the engine is running, then retry.');
  });
});
