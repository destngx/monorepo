import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { currentMonth } from '$lib/format';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, url }) => {
  const requested = url.searchParams.get('month') ?? '';
  const month = /^\d{4}-\d{2}$/.test(requested) ? requested : currentMonth();
  return loadOrError(async () => {
    const [report, transactions] = await Promise.all([api.budgetReport(month, { fetch }), api.transactions({ fetch })]);
    return { month, report, transactions };
  });
};
