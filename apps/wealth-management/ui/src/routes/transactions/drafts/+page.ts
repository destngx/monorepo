import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { captureAIStatus } from '$lib/capture/aiStatus';
import { cardTagOptions } from '$lib/capture/review';
import type { PageLoad } from './$types';

// Nothing is awaited: the header and the upload card render at once. The
// drafts (with the accounts, categories, tags and card tags their rows edit), the
// captures list and the AI status stream in on their own; a reload after a
// capture is read (invalidate 'wm:capture-drafts') keeps them on screen.
export const load: PageLoad = ({ fetch, depends }) => {
  depends('wm:capture-drafts');
  return {
    review: defer(async () => {
      const [drafts, accounts, categories, tags, cards] = await Promise.all([
        api.captureDrafts({ fetch }),
        api.accounts({ fetch }),
        api.categories({ fetch }),
        api.tags({ fetch }),
        api.cards({ fetch }),
      ]);
      return { drafts, accounts, categories, tags, cardTags: cardTagOptions(cards) };
    }),
    overview: defer(() => api.captureInbox({ fetch })),
    aiStatus: defer(() => captureAIStatus(fetch)),
  };
};
