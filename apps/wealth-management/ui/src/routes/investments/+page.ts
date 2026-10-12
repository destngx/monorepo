import { api } from '$lib/api/endpoints';
import { loadOrError } from '$lib/api/load';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
  const assets = await loadOrError(() => api.investments({ fetch }));
  // Market data is optional: holdings still render when a provider is down.
  const [usd, bankRates] = await Promise.allSettled([api.exchangeRate('USD', 'VND', fetch), api.bankRates(fetch)]);
  return {
    assets,
    usdRate: usd.status === 'fulfilled' ? usd.value : null,
    bankRates: bankRates.status === 'fulfilled' ? bankRates.value : [],
    marketError:
      usd.status === 'rejected'
        ? `USD/VND rate unavailable: ${usd.reason instanceof Error ? usd.reason.message : usd.reason}`
        : null,
  };
};
