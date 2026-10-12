import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  // The engine values holdings and totals (WM-RULE-8); market failures arrive as issues, not errors.
  const [valuation, bankRates] = await Promise.all([
    loadOrError(() => api.valuation({ fetch })),
    api.bankRates(fetch).catch(() => []),
  ]);
  return { valuation, bankRates };
};
