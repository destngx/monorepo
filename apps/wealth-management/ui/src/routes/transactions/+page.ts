import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// Nothing is awaited: the header, search and Add button render at once. The
// ledger streams into the table; the form options (accounts, categories,
// tags) stream into the Add/Edit form and the bulk bar. The card set-up says
// which card paid a row on an account billing several cards: while it loads
// or when the Cards tab is invalid (reported on the credit cards page) no
// card is named and card tags read as plain tags.
export const load: PageLoad = ({ fetch }) => ({
  transactions: defer(() => api.transactions({ fetch })),
  options: defer(async () => {
    const [accounts, categories, tags] = await Promise.all([
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
    ]);
    return { accounts, categories, tags };
  }),
  cardConfig: defer(() => api.cardPrograms({ fetch })),
});
