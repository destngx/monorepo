import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// Six calls, each streaming into its own section: the header and the Refresh
// button show at once, and a slow or failing market source holds only its
// card. The engine values holdings and totals (WM-RULE-8); market failures
// arrive as issues, not errors. Exposure and the gold/USD trends (ROADMAP M1,
// M2, M4) lead the page; fund performance (M3) compares with bank rates.
export const load: PageLoad = ({ fetch }) => ({
  valuation: defer(() => api.valuation({ fetch })),
  allocation: defer(() => api.allocation({ fetch })),
  bankRates: defer(() => api.bankRates(fetch)),
  fundPerformance: defer(() => api.fundPerformance({ fetch })),
  exposure: defer(() => api.exposure({ fetch })),
  hedges: defer(() => api.hedges({ fetch })),
});
