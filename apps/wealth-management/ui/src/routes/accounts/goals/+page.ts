import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// The header renders at once; the plan (one request) streams in with its own skeleton and error.
export const load: PageLoad = ({ fetch }) => ({ plan: defer(() => api.goalPlan({ fetch })) });
