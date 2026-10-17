import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { parseFundDetail, parseNavHistory, sourced } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

// The header (the code from the URL, the back link) shows at once; the fund
// detail streams in, and the NAV history follows it, since fmarket asks for
// the fund's id. An unknown code is the detail section's error, not a page error.
export const load: PageLoad = ({ fetch, params }) => {
  const code = params.code;
  const detail = (async () => {
    const loaded = sourced(await api.fmarket<unknown>('getProductDetails', { code }, { fetch }), parseFundDetail);
    if (!loaded.data) throw new Error(`Fund ${code} was not found on fmarket.`);
    return { fund: loaded.data, source: loaded.source };
  })();
  return {
    code,
    detail: defer(detail),
    // fmarket returns the full NAV history whatever the period, so ranges are sliced client-side.
    nav: defer(async () => {
      const { fund } = await detail;
      return sourced(await api.fmarket<unknown>('getNavHistory', { productId: fund.id }, { fetch }), parseNavHistory);
    }),
  };
};
