import { error } from '@sveltejs/kit';
import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import { allCards } from '$lib/cards/cards';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch, params }) => {
  const banks = await loadOrError(() => api.cards({ fetch }));
  const bank = banks.find((candidate) => candidate.cards.some((card) => card.name === params.name));
  const card = bank?.cards.find((candidate) => candidate.name === params.name);
  if (!bank || !card) error(404, { message: `No credit card named ${params.name}.` });
  return { bank, card, cardKeywords: allCards(banks).map((other) => other.keyword) };
};
