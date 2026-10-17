import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// Nothing is awaited: the header renders at once; the banks and their cards
// stream in, with the best-card form and programme comparison once there are cards.
export const load: PageLoad = ({ fetch }) => ({ banks: defer(() => api.cards({ fetch })) });
