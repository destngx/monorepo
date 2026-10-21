import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const [people, members] = await Promise.all([session.api.people(fetch), session.api.members(fetch)]);
  return { people, members };
};
