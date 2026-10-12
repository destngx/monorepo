import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { currentMonth } from '$lib/format';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [summary, budget] = await Promise.all([
      api.dashboard({ fetch }),
      api.budgetReport(currentMonth(), { fetch }),
    ]);
    return { summary, budget };
  });
