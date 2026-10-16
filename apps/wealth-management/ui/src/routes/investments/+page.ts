import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { settle } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  // The engine values holdings and totals (WM-RULE-8); market failures arrive as issues, not errors.
  // The allocation and bank rates settle on their own, so a failure shows on their section only.
  const [valuation, allocation, bankRates] = await Promise.all([
    loadOrError(() => api.valuation({ fetch })),
    settle(() => api.allocation({ fetch })),
    settle(() => api.bankRates(fetch)),
  ]);
  return { valuation, allocation, bankRates };
};
