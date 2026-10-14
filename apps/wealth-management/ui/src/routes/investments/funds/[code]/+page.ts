import { error } from '@sveltejs/kit';
import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { parseFundDetail, parseNavHistory, settle, sourced } from '$lib/investments/fmarket';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
  const loaded = await loadOrError(async () =>
    sourced(await api.fmarket<unknown>('getProductDetails', { code: params.code }, { fetch }), parseFundDetail),
  );
  const detail = loaded.data;
  if (!detail) error(404, { message: `Fund ${params.code} was not found on fmarket.` });
  // fmarket returns the full NAV history whatever the period, so ranges are sliced client-side.
  const nav = await settle(async () =>
    sourced(await api.fmarket<unknown>('getNavHistory', { productId: detail.id }, { fetch }), parseNavHistory),
  );
  return { detail, detailSource: loaded.source, nav };
};
