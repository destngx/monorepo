import { afterEach, describe, expect, it, vi } from 'vitest';
import { toasts } from '$lib/state/toasts.svelte';
import { ApiError } from './client';
import { classifyFailure, defer, settle } from './load';

describe('settle', () => {
  it('keeps the value of a part that loaded', async () => {
    expect(await settle(Promise.resolve(3))).toEqual({ ok: true, value: 3 });
  });

  it('keeps the error of a part that failed instead of failing the page', async () => {
    expect(await settle(Promise.reject(new Error('engine down')))).toEqual({ ok: false, error: 'engine down' });
  });
});

describe('classifyFailure', () => {
  it('Given an expired Google token When classified Then it is an auth problem with re-auth steps', () => {
    const failure = classifyFailure(new ApiError(401, 'token expired', 'OAUTH_EXPIRED', '/dashboard'));
    expect(failure).toMatchObject({ kind: 'auth', sheetsAuth: 'expired', message: 'token expired', status: 401 });
  });

  it('Given Sheets not configured When classified Then it is not-configured with setup steps', () => {
    const failure = classifyFailure(new ApiError(503, 'not configured', 'SHEETS_NOT_CONFIGURED'));
    expect(failure).toMatchObject({ kind: 'not-configured', sheetsAuth: 'missing' });
  });

  it('Given an older engine answering 404 without a code When classified Then Sheets is not configured', () => {
    const failure = classifyFailure(new ApiError(404, 'Not Found'));
    expect(failure.kind).toBe('not-configured');
    expect(failure.sheetsAuth).toBe('missing');
    expect(failure.message).toMatch(/Google Sheets is not configured/);
  });

  it('Given a wrong API token When classified Then it is an auth problem without Sheets steps', () => {
    const failure = classifyFailure(new ApiError(401, 'API token missing or wrong.', 'UNAUTHORIZED'));
    expect(failure).toMatchObject({ kind: 'auth', sheetsAuth: null });
  });

  it('Given an optional service that is down When classified Then it is unavailable with a hint', () => {
    const failure = classifyFailure(new ApiError(503, 'no provider', 'UNAVAILABLE', '/ai/briefing'));
    expect(failure.kind).toBe('unavailable');
    expect(failure.hint).toMatch(/AI is not configured/);
  });

  it('Given a network failure When classified Then the engine is unreachable', () => {
    const failure = classifyFailure(new TypeError('Failed to fetch'));
    expect(failure).toMatchObject({ kind: 'unreachable', message: 'Engine unreachable: Failed to fetch' });
  });

  it('Given any other error When classified Then it keeps the message', () => {
    expect(classifyFailure(new ApiError(500, 'boom', 'INTERNAL'))).toMatchObject({ kind: 'other', message: 'boom' });
    expect(classifyFailure('plain')).toMatchObject({ kind: 'other', message: 'plain' });
  });
});

describe('defer', () => {
  afterEach(() => {
    toasts.sheetsAuthProblem = null;
  });

  it('Given a request that succeeds When deferred Then it resolves to the value', async () => {
    expect(await defer(Promise.resolve({ total: 1 }))).toEqual({ ok: true, value: { total: 1 } });
  });

  it('Given a request that fails When deferred Then it resolves to the classified failure', async () => {
    const result = await defer(() => Promise.reject(new ApiError(500, 'boom', 'INTERNAL')));
    expect(result).toEqual({
      ok: false,
      error: 'boom',
      failure: { kind: 'other', message: 'boom', sheetsAuth: null, hint: null, status: 500 },
    });
  });

  it('Given a loader that throws synchronously When deferred Then the throw becomes a failure', async () => {
    const result = await defer(() => {
      throw new Error('bad input');
    });
    expect(result).toMatchObject({ ok: false, error: 'bad input' });
  });

  it('Given a failing request When deferred and never awaited Then no unhandled rejection leaks', async () => {
    const unhandled = vi.fn();
    process.on('unhandledRejection', unhandled);
    try {
      void defer(Promise.reject(new Error('ignored')));
      await new Promise((resolve) => setTimeout(resolve, 10));
      expect(unhandled).not.toHaveBeenCalled();
    } finally {
      process.off('unhandledRejection', unhandled);
    }
  });

  it('Given a Sheets credential failure When deferred Then the re-auth alert is raised once for the page', async () => {
    await defer(Promise.reject(new ApiError(401, 'token expired', 'OAUTH_EXPIRED')));
    expect(toasts.sheetsAuthProblem).toEqual({ message: 'token expired', kind: 'expired' });
  });

  it('Given an optional service failure When deferred Then the re-auth alert stays down', async () => {
    await defer(Promise.reject(new ApiError(503, 'down', 'UNAVAILABLE', '/market')));
    expect(toasts.sheetsAuthProblem).toBeNull();
  });
});
