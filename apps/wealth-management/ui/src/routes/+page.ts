import { api } from '$lib/api/endpoints';
import { loadOrError, settle } from '$lib/api/load';
import { loadVndRates } from '$lib/rates';
import type { PageLoad } from './$types';

// The inbox and safe-to-spend fail on their own: the rest of the home page
// still shows, with the error in their place.
export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [summary, usd, needsYou, safeToSpend] = await Promise.all([
      api.dashboard({ fetch }),
      loadVndRates(['USD'], fetch),
      settle(api.needsYou({ fetch })),
      settle(api.safeToSpend({ fetch })),
    ]);
    return { summary, usd, needsYou, safeToSpend };
  });
