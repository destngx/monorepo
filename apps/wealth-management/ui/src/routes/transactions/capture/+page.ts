import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [accounts, categories, tags, aiStatus] = await Promise.all([
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
      // AI routes are absent when AI is off; the page explains instead of failing.
      api.aiStatus(fetch).catch(() => null),
    ]);
    return { accounts, categories, tags, aiStatus };
  });
