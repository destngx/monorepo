import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { currentMonth } from '$lib/format';
import type { PageLoad } from './$types';

// The month comes from the URL, so the header and month switcher render at
// once; the report, spending windows, transactions and categories stream in
// on their own. The spending windows end today whatever the month.
export const load: PageLoad = ({ fetch, url }) => {
  const requested = url.searchParams.get('month') ?? '';
  const month = /^\d{4}-\d{2}$/.test(requested) ? requested : currentMonth();
  return {
    month,
    report: defer(() => api.budgetReport(month, { fetch })),
    spending: defer(() => api.spendingWindows({ fetch })),
    transactions: defer(() => api.transactions({ fetch })),
    categories: defer(() => api.categories({ fetch })),
  };
};
