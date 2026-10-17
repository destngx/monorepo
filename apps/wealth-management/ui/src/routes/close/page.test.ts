import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import { monthlyClose } from '$lib/close/close.testdata';
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

import ClosePage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Monthly close page', () => {
  afterEach(() => mask.hide());

  it('Given the checklist is still loading When the page renders Then the header and month links are usable at once', () => {
    render(ClosePage, { data: { engine: never(), month: '2026-08', close: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Monthly close' })).toBeTruthy();
    const current = screen.getByRole('link', { name: 'Aug 2026' });
    expect(current.getAttribute('aria-current')).toBe('page');
    expect(current.getAttribute('href')).toBe('/ui/close?month=2026-08');
    expect(screen.getByText('Loading the close checklist…')).toBeTruthy();
  });

  it('Given no month in the URL When the checklist answers Then the month the engine picked is marked current', async () => {
    render(ClosePage, {
      data: { engine: never(), month: '', close: settled(monthlyClose({ month: '2026-07' })) } as never,
    });
    await flush();

    expect(screen.getByRole('link', { name: 'Jul 2026' }).getAttribute('aria-current')).toBe('page');
  });

  it('Given the checklist fails When it settles Then its error and Retry show and the month links stay', async () => {
    render(ClosePage, {
      data: {
        engine: never(),
        month: '2026-10',
        close: defer(Promise.reject(new Error('close unavailable'))),
      } as never,
    });
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('Could not load the close checklist');
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Oct 2026' })).toBeTruthy();
  });

  it('Given the checklist has loaded When masked Then its figures stay hidden until toggled', async () => {
    render(ClosePage, { data: { engine: never(), month: '2026-10', close: settled(monthlyClose()) } as never });
    await flush();

    expect(document.body.textContent).toContain(`Over by ${MASK}`);
    expect(document.body.textContent).not.toMatch(/3\.000\.000/);
    mask.toggle();
    await flush();
    expect(document.body.textContent).toContain('3.000.000');
  });

  it('Given the snapshot item When Write the snapshot is pressed Then the button stays busy until the engine answers', async () => {
    render(ClosePage, { data: { engine: never(), month: '2026-10', close: settled(monthlyClose()) } as never });
    await flush();
    calls.length = 0;

    const write = screen.getByRole('button', { name: 'Write the snapshot' });
    await fireEvent.click(write);
    await flush();

    expect(calls).toEqual([{ name: 'writeSnapshot', args: ['2026-10'] }]);
    expect(write.getAttribute('aria-busy')).toBe('true');
    expect(write.textContent).toContain('Writing…');
  });
});
