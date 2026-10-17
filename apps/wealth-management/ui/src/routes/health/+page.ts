import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import type { PageLoad } from './$types';

// The score and the net worth history stream in on their own: the history
// reads every snapshot, so it never holds the score (or the other way round).
export const load: PageLoad = ({ fetch }) => ({
  health: defer(() => api.financialHealth({ fetch })),
  netWorthHistory: defer(() => api.netWorthHistory({ fetch })),
});
