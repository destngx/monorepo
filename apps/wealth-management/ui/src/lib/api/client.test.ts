import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, buildPath, request } from './client';
import { apiToken } from '$lib/state/settings.svelte';

function respond(status: number, body: unknown): typeof fetch {
  return vi.fn(() => Promise.resolve(new Response(typeof body === 'string' ? body : JSON.stringify(body), { status })));
}

afterEach(() => {
  apiToken.value = '';
});

async function failure(promise: Promise<unknown>): Promise<ApiError> {
  try {
    await promise;
  } catch (error) {
    if (error instanceof ApiError) return error;
    throw error;
  }
  throw new Error('expected the request to fail');
}

describe('buildPath', () => {
  it('drops empty query values', () => {
    expect(buildPath('/budget', { year: 2026, force: undefined, month: '' })).toBe('/api/budget?year=2026');
  });
});

describe('request', () => {
  it('sends the token on writes only', async () => {
    apiToken.value = 'secret';
    const fetcher = respond(200, { ok: true });

    await request('/sync', { method: 'POST', fetch: fetcher });
    await request('/accounts', { fetch: fetcher });

    const calls = vi.mocked(fetcher).mock.calls;
    expect(new Headers(calls[0][1]?.headers).get('Authorization')).toBe('Bearer secret');
    expect(new Headers(calls[1][1]?.headers).get('Authorization')).toBeNull();
  });

  it('surfaces the server error message and status', async () => {
    const error = await failure(
      request('/transactions/row-9', {
        method: 'DELETE',
        fetch: respond(409, { error: 'conflict: row changed' }),
      }),
    );

    expect(error).toBeInstanceOf(ApiError);
    expect(error.status).toBe(409);
    expect(error.message).toBe('conflict: row changed');
  });

  it('explains a missing token', async () => {
    const error = await failure(request('/sync', { method: 'POST', fetch: respond(401, '') }));
    expect(error.message).toMatch(/Settings/);
  });

  it('flags Google credential failures', async () => {
    const error = await failure(
      request('/accounts', {
        fetch: respond(500, { error: 'oauth2: "invalid_grant" "Token has been expired or revoked."' }),
      }),
    );
    expect(error.authProblem).toBe(true);
  });
});
