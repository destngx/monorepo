import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { action } from './action.svelte';
import { TOAST_MS, toasts } from './toasts.svelte';

const shown = () => toasts.items.map((toast) => [toast.kind, toast.message]);

describe('toasts', () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => {
    vi.useRealTimers();
    toasts.items = [];
    toasts.setErrorToast(null);
  });

  it('Given a failed action When reported Then the toast names what was being done', () => {
    toasts.error(new Error('disk full'), 'Could not save');
    expect(shown()).toEqual([['error', 'Could not save: disk full']]);
  });

  it('Given a toast When its time is up Then it goes away by itself', () => {
    toasts.success('Saved.');
    vi.advanceTimersByTime(TOAST_MS);
    expect(toasts.items).toEqual([]);
  });

  it('Given an app error hook When an error is reported Then the app decides the toast', () => {
    toasts.setErrorToast((error, text) =>
      error instanceof RangeError ? null : { kind: 'warning', message: `${text} Try later.` },
    );
    toasts.error(new RangeError('handled elsewhere'));
    toasts.error(new Error('AI is off'), 'Could not explain');
    expect(shown()).toEqual([['warning', 'Could not explain: AI is off Try later.']]);
  });
});

describe('action', () => {
  afterEach(() => {
    toasts.items = [];
  });

  it('Given a running action When run again Then the second run is ignored', async () => {
    let finish!: () => void;
    const fn = vi.fn(() => new Promise<void>((done) => (finish = done)));
    const save = action(fn, { success: 'Saved.' });

    const first = save.run();
    expect(save.busy).toBe(true);
    expect(await save.run()).toBe(false);
    finish();
    expect(await first).toBe(true);
    expect(fn).toHaveBeenCalledOnce();
    expect(save.busy).toBe(false);
    expect(shown()).toEqual([['success', 'Saved.']]);
  });

  it('Given an action that fails When run Then it resolves false and toasts the error', async () => {
    const save = action(() => Promise.reject(new Error('locked')), { errorContext: 'Could not save' });
    expect(await save.run()).toBe(false);
    expect(shown()).toEqual([['error', 'Could not save: locked']]);
  });
});
