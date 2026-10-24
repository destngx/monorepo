import { afterEach, describe, expect, it, vi } from 'vitest';
import { agentPath, api, ApiError } from './api';

afterEach(() => vi.unstubAllGlobals());

function respond(status: number, body: string) {
  const fetch = vi.fn().mockResolvedValue(new Response(status === 204 ? null : body, { status }));
  vi.stubGlobal('fetch', fetch);
  return fetch;
}

describe('agentPath', () => {
  it('encodes the colon in a pane id', () => {
    expect(agentPath('wE:p7N', 'screen')).toBe('/api/agents/wE%3Ap7N/screen');
  });
});

describe('api', () => {
  it('posts a prompt as JSON', async () => {
    const fetch = respond(204, '');
    await api.prompt('w1:p1', 'run the tests');
    expect(fetch).toHaveBeenCalledWith(
      '/api/agents/w1%3Ap1/prompt',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: '{"text":"run the tests"}',
      }),
    );
  });

  it("raises the engine's message and code", async () => {
    respond(409, '{"error":"The agent is waiting for an answer.","code":"AGENT_BLOCKED"}');
    const error = await api.keys('w1:p1', ['1']).catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect(error).toMatchObject({ status: 409, code: 'AGENT_BLOCKED', message: 'The agent is waiting for an answer.' });
  });

  it('explains a proxy error when the engine is down', async () => {
    respond(502, 'Bad Gateway');
    await expect(api.screen('w1:p1')).rejects.toMatchObject({ message: 'The engine is not running on the laptop.' });
  });

  it('explains a network failure', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Load failed')));
    await expect(api.screen('w1:p1')).rejects.toMatchObject({ code: 'UNREACHABLE' });
  });
});
