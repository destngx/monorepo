import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// Nothing is awaited: the header and New item render at once. The bills
// report (safe to spend, calendar, suggestions, items) and the idle cash
// check stream into their own sections.
export const load: PageLoad = ({ fetch }) => ({
  bills: defer(() => api.bills({ fetch })),
  idleCash: defer(() => api.idleCash({ fetch })),
});
