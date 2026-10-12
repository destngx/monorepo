import { error } from '@sveltejs/kit';
import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { findCard } from '$lib/cards/cards';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
  const banks = await loadOrError(() => api.cards({ fetch }));
  const card = findCard(banks, params.name);
  if (!card) error(404, { message: `No credit card named ${params.name}.` });
  return { card };
};
