import { api } from '$lib/api/endpoints';
import type { PageLoad } from './$types';

// AI routes are absent when AI is off; the page explains instead of failing.
export const load: PageLoad = async ({ fetch }) => ({ aiStatus: await api.aiStatus(fetch).catch(() => null) });
