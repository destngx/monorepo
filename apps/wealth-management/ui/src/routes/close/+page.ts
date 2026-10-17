import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// Without ?month= the engine picks the close month: last month during the first
// days of a month. The header and month links render at once; the checklist
// streams in with its own skeleton and error.
export const load: PageLoad = ({ fetch, url }) => {
  const requested = url.searchParams.get('month') ?? '';
  const month = /^\d{4}-\d{2}$/.test(requested) ? requested : '';
  return { month, close: defer(() => api.monthlyClose(month, { fetch })) };
};
