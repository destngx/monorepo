import { auth } from '$lib/api/endpoints';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => ({ status: await auth.setupStatus(fetch) });
