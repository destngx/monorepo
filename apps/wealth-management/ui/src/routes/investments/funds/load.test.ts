import { describe, expect, it, vi } from 'vitest';
import type { Streamed } from '$lib/api/load';
import type { FundType } from '$lib/investments/fmarket';

const fmarket = vi.hoisted(() => vi.fn());
const bankRates = vi.hoisted(() => vi.fn());
vi.mock('$lib/api/endpoints', () => ({ api: { fmarket, bankRates } }));

import { load } from './+page';

describe('Fmarket funds load', () => {
  it('Given fmarket fails for one fund type When loading Then the load returns at once and only that list fails', async () => {
    const pending: ((value: unknown) => void)[] = [];
    fmarket.mockImplementation((_action: string, params: { isMMFFund?: boolean }) =>
      params.isMMFFund ? Promise.reject(new Error('fmarket down')) : new Promise((resolve) => pending.push(resolve)),
    );
    bankRates.mockResolvedValue({ rates: [] });

    const result = load({ fetch } as never);
    expect(result).not.toBeInstanceOf(Promise);
    const data = result as unknown as { funds: Record<FundType, Streamed<unknown>>; bankRates: Streamed<unknown> };

    expect(data.funds.STOCK).toBeInstanceOf(Promise);
    expect(await data.funds.MMF).toMatchObject({ ok: false, error: 'fmarket down' });
    expect(await data.bankRates).toMatchObject({ ok: true });
    pending.forEach((resolve) => resolve({ data: { rows: [] } }));
    expect(await data.funds.STOCK).toMatchObject({ ok: true, value: { data: [] } });
  });
});
