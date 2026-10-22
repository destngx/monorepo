import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  return { overview: await session.api.personMetrics(params.id, fetch) };
};
