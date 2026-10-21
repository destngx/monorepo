import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { captureAIStatus } from '$lib/capture/aiStatus';
import { cardNameOptions, cardTagOptions } from '$lib/capture/review';
import type { PageLoad } from './$types';

// Nothing is awaited: materials can be added while the AI status and the
// review options (accounts, categories, tags, card tags) stream in. Analyze waits for
// the AI status; the review waits for the options.
export const load: PageLoad = ({ fetch }) => ({
  options: defer(async () => {
    const [accounts, categories, tags, cards] = await Promise.all([
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
      api.cards({ fetch }),
    ]);
    return { accounts, categories, tags, cardTags: cardTagOptions(cards), cardNames: cardNameOptions(cards) };
  }),
  aiStatus: defer(() => captureAIStatus(fetch)),
});
