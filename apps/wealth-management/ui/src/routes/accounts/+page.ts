import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

// 366 days covers the 12-month balance history (WM-ACC-8); sparklines use the last 60.
const HISTORY_DAYS = 366;

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [accounts, histories, summary] = await Promise.all([
      api.accounts({ fetch }),
      api.accountHistories(HISTORY_DAYS, { fetch }),
      api.dashboard({ fetch }),
    ]);
    return { accounts, histories, balanceSheet: summary.balanceSheet };
  });
