import { describe, expect, it, vi } from 'vitest';
import { classifyFailure, createLoader, defer, settle, settled, UNREACHABLE_HINT, type LoadFailure } from './load';

/** An engine error as the apps' ApiError carries it. */
const apiError = (status: number, message: string, code?: string) =>
  Object.assign(new Error(message), { status, code });

describe('classifyFailure', () => {
  it('Given a 401 or 403 When classified Then it is an auth problem', () => {
    expect(classifyFailure(apiError(401, 'API token missing or wrong.'))).toMatchObject({ kind: 'auth', status: 401 });
    expect(classifyFailure(apiError(403, 'Admins only.')).kind).toBe('auth');
  });

  it('Given a not-configured code When classified Then it is not-configured, whatever the status', () => {
    expect(classifyFailure(apiError(503, 'not set up', 'SHEETS_NOT_CONFIGURED')).kind).toBe('not-configured');
    expect(classifyFailure(apiError(409, 'not set up', 'NOT_CONFIGURED')).kind).toBe('not-configured');
  });

  it('Given an optional service that is off When classified Then it is unavailable', () => {
    expect(classifyFailure(apiError(503, 'no provider', 'UNAVAILABLE')).kind).toBe('unavailable');
  });

  it('Given a fetch that never reached the engine When classified Then it is unreachable with what to do', () => {
    expect(classifyFailure(new TypeError('Failed to fetch'))).toEqual({
      kind: 'unreachable',
      message: 'Engine unreachable: Failed to fetch',
      hint: UNREACHABLE_HINT,
    });
  });

  it('Given anything else When classified Then it is other with its message', () => {
    expect(classifyFailure(apiError(500, 'engine broke', 'INTERNAL'))).toMatchObject({ kind: 'other', status: 500 });
    expect(classifyFailure('plain text')).toEqual({ kind: 'other', message: 'plain text', hint: null });
  });
});

describe('defer', () => {
  it('Given a request that loads When settled Then the value is kept', async () => {
    expect(await defer(Promise.resolve(3))).toEqual({ ok: true, value: 3 });
  });

  it('Given a request that fails When settled Then it resolves with the classified failure, never rejects', async () => {
    const result = await defer(() => Promise.reject(apiError(401, 'expired')));
    expect(result).toMatchObject({ ok: false, error: 'expired', failure: { kind: 'auth' } });
  });

  it('Given a loader that throws before it returns a promise When deferred Then that failure is kept too', async () => {
    const result = await defer(() => {
      throw new Error('bad input');
    });
    expect(result).toMatchObject({ ok: false, error: 'bad input' });
  });
});

describe('createLoader', () => {
  it("Given an app classifier and hook When a request fails Then the app's failure is used and reported", async () => {
    const failure: LoadFailure = { kind: 'auth', message: 'Sign in to Google again.', hint: 'Follow the steps above.' };
    const onFailure = vi.fn();
    const loader = createLoader({ classify: () => failure, onFailure });
    const cause = new Error('token expired');

    expect(await loader.defer(Promise.reject(cause))).toEqual({ ok: false, error: failure.message, failure });
    expect(onFailure).toHaveBeenCalledWith(failure, cause);
  });
});

describe('settle and settled', () => {
  it('keeps the value of a part that loaded', async () => {
    expect(await settle(Promise.resolve(3))).toEqual({ ok: true, value: 3 });
    expect(await settled(4)).toEqual({ ok: true, value: 4 });
  });

  it('keeps the error of a part that failed instead of failing the page', async () => {
    expect(await settle(Promise.reject(new Error('engine down')))).toEqual({ ok: false, error: 'engine down' });
  });
});
