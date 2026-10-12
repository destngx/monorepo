import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, buildPath, request, sheetsAuthKind, unavailableHint } from './client';
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

describe('error codes', () => {
  it('keeps the envelope code', async () => {
    const error = await failure(
      request('/transactions/row-9', {
        method: 'PATCH',
        fetch: respond(409, { error: 'row changed', code: 'CONFLICT' }),
      }),
    );
    expect(error.code).toBe('CONFLICT');
    expect(error.authProblem).toBe(false);
  });

  it('classifies Google credential codes without reading the message', async () => {
    const missing = await failure(
      request('/accounts', {
        fetch: respond(503, { error: 'GOOGLE_SHEETS_ID is not set', code: 'SHEETS_NOT_CONFIGURED' }),
      }),
    );
    expect(missing.sheetsAuth).toBe('missing');
    const expired = await failure(
      request('/accounts', { fetch: respond(502, { error: 'Google said no', code: 'OAUTH_EXPIRED' }) }),
    );
    expect(expired.sheetsAuth).toBe('expired');
  });

  it('trusts a code over a message that looks like an auth failure', () => {
    expect(sheetsAuthKind('UPSTREAM', 'oauth2: invalid_grant')).toBeNull();
  });

  it('treats UNAVAILABLE as a service problem, not a Google one', async () => {
    const ai = await failure(
      request('/ai/briefing', {
        method: 'POST',
        fetch: respond(503, { error: 'AI_API_KEY not set', code: 'UNAVAILABLE' }),
      }),
    );
    expect(ai.authProblem).toBe(false);
    expect(ai.path).toBe('/ai/briefing');
    expect(ai.unavailableHint).toMatch(/Settings > AI/);
    expect(unavailableHint('/market/pulse')).not.toMatch(/AI/);
    const other = await failure(request('/accounts', { fetch: respond(500, { error: 'boom', code: 'INTERNAL' }) }));
    expect(other.unavailableHint).toBeNull();
  });

  it('falls back to the message for responses without a code', () => {
    expect(sheetsAuthKind(undefined, 'oauth2: "invalid_grant"')).toBe('expired');
    expect(sheetsAuthKind(undefined, 'missing Google Sheets configuration: GOOGLE_SHEETS_ID')).toBe('missing');
    expect(sheetsAuthKind(undefined, 'row not found')).toBeNull();
  });
});
