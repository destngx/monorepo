import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [transactions, accounts, categories, tags, cards] = await Promise.all([
      api.transactions({ fetch }),
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
      // Only colours card tags: an invalid card tab (reported on the credit
      // cards page) must not block the ledger, so tags then stay neutral.
      api.cardPrograms({ fetch }).catch(() => null),
    ]);
    const cardKeywords = cards?.cards.map((card) => card.keyword) ?? [];
    return { transactions, accounts, categories, tags, cardKeywords };
  });
