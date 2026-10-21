import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import type { VisitFilter, VisitType } from '$lib/api/types';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent, url }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const filter: VisitFilter = {
    type: (url.searchParams.get('type') ?? '') as VisitType | '',
    providerId: url.searchParams.get('providerId') ?? '',
    from: url.searchParams.get('from') ?? '',
    to: url.searchParams.get('to') ?? '',
  };
  const api = session.api;
  const [person, visits, providers, people] = await Promise.all([
    api.person(params.id, fetch),
    api.visits(params.id, filter, fetch),
    api.providers(fetch),
    api.people(fetch),
  ]);
  return { person, visits, providers, people, filter };
};
