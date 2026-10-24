import { describe, expect, it, vi } from 'vitest';
import type { Streamed } from '$lib/api/load';

const api = vi.hoisted(() => ({ needsYou: vi.fn(), dashboard: vi.fn(), safeToSpend: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api }));
vi.mock('$lib/rates', () => ({
  loadVndRates: vi.fn(() => Promise.resolve({ rates: { VND: 1 }, quotes: {}, errors: [] })),
}));

import { load } from './+page';

type HomeData = { needsYou: Streamed<unknown>; summary: Streamed<unknown>; safeToSpend: Streamed<unknown> };

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('Home load', () => {
  it('Given what needs you is loading When the page loads Then the other cards ask only once it has settled', async () => {
    let answer!: (value: unknown) => void;
    api.needsYou.mockReturnValue(new Promise((resolve) => (answer = resolve)));
    api.dashboard.mockResolvedValue({});
    api.safeToSpend.mockResolvedValue({});

    const data = load({ fetch } as never) as unknown as HomeData;
    await flush();
    expect(api.needsYou).toHaveBeenCalledOnce();
    expect(api.dashboard).not.toHaveBeenCalled();
    expect(api.safeToSpend).not.toHaveBeenCalled();

    answer({ items: [] });
    expect(await data.summary).toMatchObject({ ok: true });
    expect(await data.safeToSpend).toMatchObject({ ok: true });
  });

  it('Given what needs you fails When the page loads Then the other cards still load', async () => {
    api.needsYou.mockRejectedValue(new Error('sheet down'));
    api.dashboard.mockResolvedValue({});
    api.safeToSpend.mockResolvedValue({});

    const data = load({ fetch } as never) as unknown as HomeData;

    expect(await data.needsYou).toMatchObject({ ok: false });
    expect(await data.summary).toMatchObject({ ok: true });
  });
});
