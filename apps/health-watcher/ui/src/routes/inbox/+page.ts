import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const api = session.api;
  const [items, people, providers, groups] = await Promise.all([
    api.inbox(fetch),
    api.people(fetch),
    api.providers(fetch),
    api.groups(fetch),
  ]);
  return { items, people, providers, groups };
};
