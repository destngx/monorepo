import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// The stored list shows as soon as it is read, while the engine rules run
// (ROADMAP R7) and replace it with what they find. The refresh writes, so
// without write access the stored list stays and the page says the checks did
// not run. Neither request holds the page.
export const load: PageLoad = ({ fetch, url }) => {
  const page = url.searchParams.get('page') ?? '';
  return {
    page,
    list: defer(() => api.recommendations({ page }, { fetch })),
    refresh: defer(() => api.refreshRecommendations({ page }, fetch)),
  };
};
