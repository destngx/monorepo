import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { catalog } from '$lib/state/catalog.svelte';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const api = session.api;
  const [detail, people, providers] = await Promise.all([
    api.item(params.id, fetch),
    api.people(fetch),
    api.providers(fetch),
    catalog.load(fetch),
  ]);
  return { detail, people, providers };
};
