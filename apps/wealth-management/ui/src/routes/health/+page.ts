import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = ({ fetch }) =>
  loadOrError(async () => {
    const [health, netWorthHistory] = await Promise.all([
      api.financialHealth({ fetch }),
      api.netWorthHistory({ fetch }),
    ]);
    return { health, netWorthHistory };
  });
