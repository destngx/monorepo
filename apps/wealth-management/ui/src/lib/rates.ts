import { api } from './api/endpoints';
import type { RequestOptions } from './api/client';

/** VND per unit for each currency, plus a message for every rate that failed to load. */
export interface LoadedRates {
  rates: Record<string, number>;
  errors: string[];
}

/**
 * Loads currency-to-VND rates from the engine (WM-DASH-7, WM-ACC-8). A failed
 * rate does not fail the page: it is reported in `errors` and shown next to
 * the figures that need it.
 */
export async function loadVndRates(currencies: string[], fetch?: RequestOptions['fetch']): Promise<LoadedRates> {
  const wanted = [...new Set(currencies.filter((currency) => currency !== 'VND'))];
  const results = await Promise.allSettled(wanted.map((currency) => api.exchangeRate(currency, 'VND', fetch)));
  const loaded: LoadedRates = { rates: { VND: 1 }, errors: [] };
  results.forEach((result, i) => {
    if (result.status === 'fulfilled' && result.value.rate > 0) loaded.rates[wanted[i]] = result.value.rate;
    else {
      const reason =
        result.status === 'rejected'
          ? result.reason instanceof Error
            ? result.reason.message
            : String(result.reason)
          : 'the engine returned no rate';
      loaded.errors.push(`${wanted[i]}/VND rate unavailable: ${reason}`);
    }
  });
  return loaded;
}
