import { redirect } from '@sveltejs/kit';
import { base } from '$app/paths';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, url, parent }) => {
  await parent();
  if (!session.tenantId) redirect(307, `${base}/`);
  const q = url.searchParams.get('q')?.trim() ?? '';
  if (!q) return { q, result: null };
  return { q, result: await session.api.search(q, fetch) };
};
