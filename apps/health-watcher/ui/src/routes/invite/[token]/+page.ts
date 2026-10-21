import { auth } from '$lib/api/endpoints';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => ({ invite: await auth.invite(params.token, fetch) });
