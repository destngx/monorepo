import { describe, expect, it } from 'vitest';
import { defer, settled } from '$lib/api/load';
import { joinStreamed } from './streams';

describe('joinStreamed', () => {
  it('Given both parts load When joined Then it resolves to both values in order', async () => {
    expect(await joinStreamed(settled(1), settled('a'))).toEqual({ ok: true, value: [1, 'a'] });
  });

  it('Given one part fails When joined Then it resolves to that failure without rejecting', async () => {
    const joined = await joinStreamed(settled(1), defer(Promise.reject(new Error('histories down'))));
    expect(joined.ok).toBe(false);
    expect(!joined.ok && joined.error).toBe('histories down');
  });
});
