import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { loadVndRates } from '$lib/rates';
import type { PageLoad } from './$types';

// Every part of home streams in (docs/wealth-management/AGENTS.md "Loading and
// feedback"): the header and quick actions show at once, and each card shows
// its skeleton, then its data or its own error with a Retry.
export const load: PageLoad = ({ fetch }) => ({
  summary: defer(() => api.dashboard({ fetch })),
  usd: defer(() => loadVndRates(['USD'], fetch)),
  needsYou: defer(() => api.needsYou({ fetch })),
  safeToSpend: defer(() => api.safeToSpend({ fetch })),
});
