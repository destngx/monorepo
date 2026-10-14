import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const client = vi.hoisted(() => ({ request: vi.fn(), requestStream: vi.fn() }));
vi.mock('$lib/api/client', async (original) => ({
  ...(await original<typeof import('$lib/api/client')>()),
  request: client.request,
  requestStream: client.requestStream,
}));

import { api } from '$lib/api/endpoints';
import { AI_USAGE_SETTLE_MS, aiUsage } from './usage.svelte';

const usageCalls = () => client.request.mock.calls.filter(([path]) => path === '/ai/usage').length;

function ndjson(...lines: unknown[]): ReadableStream<Uint8Array> {
  const bytes = new TextEncoder().encode(lines.map((line) => `${JSON.stringify(line)}\n`).join(''));
  return new ReadableStream({
    start(controller) {
      controller.enqueue(bytes);
      controller.close();
    },
  });
}

async function settle() {
  await vi.advanceTimersByTimeAsync(AI_USAGE_SETTLE_MS);
}

describe('AI usage refresh after AI requests', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    client.request.mockReset();
    client.requestStream.mockReset();
    client.request.mockImplementation((path: string) =>
      Promise.resolve(
        path === '/ai/usage' ? { provider: 'openai', windows: [{ name: '5h', usedPercent: 42 }] } : { summary: 'ok' },
      ),
    );
  });

  afterEach(() => vi.useRealTimers());

  it('refetches the quota once an AI request finishes', async () => {
    await api.briefing();
    expect(usageCalls()).toBe(0);

    await settle();

    expect(usageCalls()).toBe(1);
    expect(aiUsage.usage?.windows[0].usedPercent).toBe(42);
  });

  it('refetches after a failed AI request too, since it may have spent quota', async () => {
    client.request.mockImplementation((path: string) =>
      path === '/ai/usage'
        ? Promise.resolve({ provider: 'openai', windows: [] })
        : Promise.reject(new Error('upstream failed')),
    );

    await expect(api.healthCommentary()).rejects.toThrow('upstream failed');
    await settle();

    expect(usageCalls()).toBe(1);
  });

  it('refetches once a streamed AI answer ends, not when it starts', async () => {
    client.requestStream.mockResolvedValue(ndjson({ type: 'delta', text: 'hi' }, { type: 'done' }));

    const events = api.chat([{ role: 'user', content: 'hi' }], undefined);
    await events.next();
    await settle();
    expect(usageCalls()).toBe(0);

    for await (const _ of events) void _;
    await settle();

    expect(usageCalls()).toBe(1);
  });

  it('coalesces a burst of AI requests into one refetch', async () => {
    await Promise.all([api.briefing(), api.budgetAdvice(), api.suggestCategory('Grab'), api.newsSentiment('VN')]);
    await settle();

    expect(usageCalls()).toBe(1);
  });

  it('reads the quota again when a fetch started before the AI request ended', async () => {
    let finishEarlier: ((usage: unknown) => void) | undefined;
    client.request.mockImplementationOnce(() => new Promise((resolve) => (finishEarlier = resolve)));
    const earlier = aiUsage.load();
    await api.briefing();

    await settle();
    finishEarlier?.({ provider: 'openai', windows: [{ name: '5h', usedPercent: 10 }] });
    await earlier;
    await vi.advanceTimersByTimeAsync(0);

    expect(usageCalls()).toBe(2);
    expect(aiUsage.usage?.windows[0].usedPercent).toBe(42);
  });

  it('leaves the quota alone for requests that do not call the model', async () => {
    await api.aiStatus();
    await api.aiModels();
    await api.accounts();
    await settle();

    expect(usageCalls()).toBe(0);
  });
});
