import { api } from '$lib/api/endpoints';
import { defer } from '$lib/api/load';
import { loadVndRates } from '$lib/rates';
import type { PageLoad } from './$types';

// Every part of home streams in (docs/wealth-management/AGENTS.md "Loading and
// feedback"): the header and quick actions show at once, and each card shows
// its skeleton, then its data or its own error with a Retry.
//
// What needs you loads first and alone (WM-HOME-1): it reads every tab the
// other cards read, so they ask once it has settled and find the sheet data
// cached instead of competing with it. The USD rate does not read the sheet.
export const load: PageLoad = ({ fetch }) => {
  const needsYou = defer(() => api.needsYou({ fetch }));
  const afterNeedsYou = <T>(request: () => Promise<T>) =>
    defer(async () => {
      await needsYou;
      return request();
    });
  return {
    needsYou,
    summary: afterNeedsYou(() => api.dashboard({ fetch })),
    safeToSpend: afterNeedsYou(() => api.safeToSpend({ fetch })),
    usd: defer(() => loadVndRates(['USD'], fetch)),
  };
};
