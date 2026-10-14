import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { settle } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  // The engine values holdings and totals (WM-RULE-8); market failures arrive as issues, not errors.
  // Bank rates settle on their own, so their failure is shown on their card only.
  const [valuation, bankRates] = await Promise.all([
    loadOrError(() => api.valuation({ fetch })),
    settle(() => api.bankRates(fetch)),
  ]);
  return { valuation, bankRates };
};
