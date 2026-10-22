import { beforeEach, describe, expect, it, vi } from 'vitest';
import { engine } from '$lib/api/endpoints';
import type { AIUsageView } from '$lib/api/types';
import { aiUsage } from './aiUsage.svelte';

describe('AI quota state', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('Given two gauges, when both load at once, then one request answers both', async () => {
    const view: AIUsageView = {
      enabled: true,
      usage: { provider: 'openai', windows: [{ name: '5h', usedPercent: 12 }] },
    };
    const spy = vi.spyOn(engine, 'aiUsage').mockResolvedValue(view);

    await Promise.all([aiUsage.load(), aiUsage.load()]);

    expect(spy).toHaveBeenCalledTimes(1);
    expect(aiUsage.enabled).toBe(true);
    expect(aiUsage.usage?.windows[0].usedPercent).toBe(12);
    expect(aiUsage.error).toBe('');
  });

  it('Given AI reading is off, when loaded, then there is nothing to show', async () => {
    vi.spyOn(engine, 'aiUsage').mockResolvedValue({ enabled: false });

    await aiUsage.load();

    expect(aiUsage.enabled).toBe(false);
    expect(aiUsage.usage).toBeNull();
  });

  it('Given the gateway or the engine fails, when loaded, then the reason is kept', async () => {
    vi.spyOn(engine, 'aiUsage').mockResolvedValue({ enabled: true, error: 'the AI provider returned 502 Bad Gateway' });
    await aiUsage.load();
    expect(aiUsage.error).toBe('the AI provider returned 502 Bad Gateway');

    vi.spyOn(engine, 'aiUsage').mockRejectedValue(new Error('Failed to fetch'));
    await aiUsage.load();
    expect(aiUsage.enabled).toBe(true);
    expect(aiUsage.usage).toBeNull();
    expect(aiUsage.error).toBe('Failed to fetch');
  });
});
