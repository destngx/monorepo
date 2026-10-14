import { api } from '$lib/api/endpoints';
import {
  FUND_TYPES,
  fundListParams,
  parseFundList,
  settle,
  sourced,
  type FundSummary,
  type FundType,
  type Section,
  type Sourced,
} from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  // Each section settles on its own: one failing fmarket action leaves the rest usable.
  const [lists, bankRates] = await Promise.all([
    Promise.all(
      FUND_TYPES.map(({ key }) =>
        settle(async () =>
          sourced(await api.fmarket<unknown>('getProductsFilterNav', fundListParams(key), { fetch }), parseFundList),
        ),
      ),
    ),
    settle(() => api.bankRates(fetch)),
  ]);
  const funds = {} as Record<FundType, Section<Sourced<FundSummary[]>>>;
  FUND_TYPES.forEach(({ key }, i) => (funds[key] = lists[i]));
  return { funds, bankRates };
};
