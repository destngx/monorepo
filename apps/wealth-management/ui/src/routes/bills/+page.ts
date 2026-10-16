import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) => loadOrError(async () => ({ bills: await api.bills({ fetch }) }));
