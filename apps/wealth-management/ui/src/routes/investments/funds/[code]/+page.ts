import { error } from '@sveltejs/kit';
import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { parseFundDetail, parseNavHistory, settle } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
  const detail = await loadOrError(async () =>
    parseFundDetail(await api.fmarket<unknown>('getProductDetails', { code: params.code }, { fetch })),
  );
  if (!detail) error(404, { message: `Fund ${params.code} was not found on fmarket.` });
  // fmarket returns the full NAV history whatever the period, so ranges are sliced client-side.
  const nav = await settle(async () =>
    parseNavHistory(await api.fmarket<unknown>('getNavHistory', { productId: detail.id }, { fetch })),
  );
  return { detail, nav };
};
