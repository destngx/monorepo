import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.tenantId) return { people: [], inboxCount: 0 };
  const [people, inbox] = await Promise.all([session.api.people(fetch), session.api.inbox(fetch)]);
  return { people, inboxCount: inbox.length };
};
