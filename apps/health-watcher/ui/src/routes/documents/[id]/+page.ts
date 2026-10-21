import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { catalog } from '$lib/state/catalog.svelte';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const api = session.api;
  const [detail] = await Promise.all([api.document(params.id, fetch), catalog.load(fetch)]);
  const person = await api.person(detail.document.personId, fetch);
  return { detail, person };
};
