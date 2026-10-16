import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch, depends }) => {
  depends('wm:capture-drafts');
  return loadOrError(async () => {
    const [accounts, categories, tags, drafts, overview, aiStatus] = await Promise.all([
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
      api.captureDrafts({ fetch }),
      api.captureInbox({ fetch }),
      // AI routes are absent when AI is off; the page explains instead of failing.
      api.aiStatus(fetch).catch(() => null),
    ]);
    return { accounts, categories, tags, drafts, overview, aiStatus };
  });
};
