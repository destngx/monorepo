import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { findCard } from '$lib/cards/cards';
import type { PageLoad } from './$types';

// The card name comes from the URL, so the header, back link and AI summary
// render at once; the card's report streams in. An unknown name resolves to
// null and the page says so in place of the report.
export const load: PageLoad = ({ fetch, params }) => ({
  name: params.name,
  detail: defer(async () => findCard(await api.cards({ fetch }), params.name)),
});
