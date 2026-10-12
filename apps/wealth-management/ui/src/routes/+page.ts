import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { currentMonth } from '$lib/format';
import { loadVndRates } from '$lib/rates';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [summary, budget, usd] = await Promise.all([
      api.dashboard({ fetch }),
      api.budgetReport(currentMonth(), { fetch }),
      loadVndRates(['USD'], fetch),
    ]);
    return { summary, budget, usd };
  });
