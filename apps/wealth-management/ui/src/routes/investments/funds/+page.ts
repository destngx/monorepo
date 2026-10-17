import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import {
  FUND_TYPES,
  fundListParams,
  parseFundList,
  sourced,
  type FundSummary,
  type FundType,
  type Sourced,
} from '$lib/investments/fmarket';
import type { Streamed } from '$lib/api/load';
import type { PageLoad } from './$types';

// Every fmarket list and the bank rates stream in on their own: the tabs and
// fund-type filters show at once, and one failing list holds only its table.
export const load: PageLoad = ({ fetch }) => {
  const funds = {} as Record<FundType, Streamed<Sourced<FundSummary[]>>>;
  for (const { key } of FUND_TYPES) {
    funds[key] = defer(async () =>
      sourced(await api.fmarket<unknown>('getProductsFilterNav', fundListParams(key), { fetch }), parseFundList),
    );
  }
  return { funds, bankRates: defer(() => api.bankRates(fetch)) };
};
