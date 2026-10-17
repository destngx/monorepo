import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { parseYear } from '$lib/cashflow/yearly';
import { currentMonth } from '$lib/format';
import type { PageLoad } from './$types';

// The year comes from the URL, so the header and year switcher render at
// once; the year's report streams in.
export const load: PageLoad = ({ fetch, url }) => {
  const year = parseYear(url.searchParams.get('year'));
  return {
    year: year ?? Number(currentMonth().slice(0, 4)),
    report: defer(() => api.yearlyCashFlow(year, { fetch })),
  };
};
