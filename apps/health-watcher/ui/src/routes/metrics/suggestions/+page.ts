import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { catalog } from '$lib/state/catalog.svelte';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const [suggestions] = await Promise.all([session.api.metricSuggestions(fetch), catalog.load(fetch)]);
  return { suggestions };
};
