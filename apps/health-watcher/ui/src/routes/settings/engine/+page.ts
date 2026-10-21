import { error } from '@sveltejs/kit';
import { engine } from '$lib/api/endpoints';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.me?.user.admin) error(403, 'Only an engine admin can see engine settings.');
  const [view, users] = await Promise.all([engine.settings(fetch), engine.users(fetch)]);
  return { view, users };
};
