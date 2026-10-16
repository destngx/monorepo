import { describe, expect, it } from 'vitest';
import { settle } from './load';

describe('settle', () => {
  it('keeps the value of a part that loaded', async () => {
    expect(await settle(Promise.resolve(3))).toEqual({ ok: true, value: 3 });
  });

  it('keeps the error of a part that failed instead of failing the page', async () => {
    expect(await settle(Promise.reject(new Error('engine down')))).toEqual({ ok: false, error: 'engine down' });
  });
});
