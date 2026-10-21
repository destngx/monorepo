import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const api = session.api;
  const visit = await api.visit(params.id, fetch);
  const [person, people, providers] = await Promise.all([
    api.person(visit.personId, fetch),
    api.people(fetch),
    api.providers(fetch),
  ]);
  return { visit, person, people, providers };
};
