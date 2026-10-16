import { api } from '$lib/api/endpoints';
import { ApiError } from '$lib/api/client';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

// Reading the list runs the engine rules first (ROADMAP R7). The refresh
// writes, so without write access the stored list is shown and the page says
// the checks did not run.
export const load: PageLoad = ({ fetch, url }) =>
  loadOrError(async () => {
    const page = url.searchParams.get('page') ?? '';
    try {
      return { list: await api.refreshRecommendations({ page }, fetch), page, refreshed: true };
    } catch (error) {
      if (!(error instanceof ApiError) || (error.status !== 401 && error.status !== 403)) throw error;
      return { list: await api.recommendations({ page }, { fetch }), page, refreshed: false };
    }
  });
