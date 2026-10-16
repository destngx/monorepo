import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

// Without ?month= the engine picks the close month: last month during the first days of a month.
export const load: PageLoad = ({ fetch, url }) => {
  const requested = url.searchParams.get('month') ?? '';
  const month = /^\d{4}-\d{2}$/.test(requested) ? requested : '';
  return loadOrError(async () => ({ close: await api.monthlyClose(month, { fetch }) }));
};
