import { api } from './api/endpoints';
import type { RequestOptions } from './api/client';
import type { ExchangeRate, RateBasis } from './api/types';
import { formatDate } from './format';

/**
 * VND per unit for each currency, the quote behind each rate (for its
 * source), plus a message for every rate that failed to load.
 */
export interface LoadedRates {
  rates: Record<string, number>;
  quotes: Record<string, ExchangeRate>;
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
  const loaded: LoadedRates = { rates: { VND: 1 }, quotes: {}, errors: [] };
  results.forEach((result, i) => {
    if (result.status === 'fulfilled' && result.value.rate > 0) {
      loaded.rates[wanted[i]] = result.value.rate;
      loaded.quotes[wanted[i]] = result.value;
    } else {
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

const BASIS_LABELS: Record<RateBasis, string> = {
  bank_sell: 'bank sell',
  market: 'market',
  reference: 'daily reference',
  crypto_market: 'crypto market',
};

/**
 * The SourceBadge detail of a rate: what it quotes and the provider's date,
 * e.g. "bank sell · 14/10/2026". Undefined when neither is known.
 */
export function rateDetail(rate: ExchangeRate): string | undefined {
  const parts = [rate.basis ? BASIS_LABELS[rate.basis] : '', rate.timestamp ? formatDate(rate.timestamp) : ''];
  return parts.filter(Boolean).join(' · ') || undefined;
}
