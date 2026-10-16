import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { parseYear } from '$lib/cashflow/yearly';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, url }) =>
  loadOrError(async () => ({ report: await api.yearlyCashFlow(parseYear(url.searchParams.get('year')), { fetch }) }));
