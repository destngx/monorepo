import type { Invite, UploadToken } from '$lib/api/types';
import { session } from '$lib/state/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, parent }) => {
  await parent();
  if (!session.tenantId) return { members: [], invites: [] as Invite[], tokens: [] as UploadToken[] };
  const api = session.api;
  const [members, invites, tokens] = await Promise.all([
    api.members(fetch),
    session.canManage ? api.invites(fetch) : Promise.resolve([] as Invite[]),
    session.canEdit ? api.uploadTokens(fetch) : Promise.resolve([] as UploadToken[]),
  ]);
  return { members, invites, tokens };
};
