import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [transactions, accounts, categories, tags] = await Promise.all([
      api.transactions({ fetch }),
      api.accounts({ fetch }),
      api.categories({ fetch }),
      api.tags({ fetch }),
    ]);
    return { transactions, accounts, categories, tags };
  });
