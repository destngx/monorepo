import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { settle } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  // The engine values holdings and totals (WM-RULE-8); market failures arrive as issues, not errors.
  // The allocation and bank rates settle on their own, so a failure shows on their section only.
  // Fund performance (ROADMAP M3) also settles on its own: NAV history or bank rates may be down.
  // Exposure and the gold/USD trends (ROADMAP M1, M2, M4) lead the page and settle on their own too.
  const [valuation, allocation, bankRates, fundPerformance, exposure, hedges] = await Promise.all([
    loadOrError(() => api.valuation({ fetch })),
    settle(() => api.allocation({ fetch })),
    settle(() => api.bankRates(fetch)),
    settle(() => api.fundPerformance({ fetch })),
    settle(() => api.exposure({ fetch })),
    settle(() => api.hedges({ fetch })),
  ]);
  return { valuation, allocation, bankRates, fundPerformance, exposure, hedges };
};
