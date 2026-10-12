import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [accounts, histories, summary] = await Promise.all([
      api.accounts({ fetch }),
      api.accountHistories(60, { fetch }),
      api.dashboard({ fetch }),
    ]);
    return { accounts, histories, balanceSheet: summary.balanceSheet };
  });
