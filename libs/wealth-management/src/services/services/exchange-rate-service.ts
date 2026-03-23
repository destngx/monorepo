import { getCached, setCache } from '@wealth-management/utils';
import { NetworkError, isAppError, getErrorMessage } from '../../utils/errors';

const CACHE_KEY = 'exchange-rate:usdt-vnd';
const COINGECKO_API = 'https://api.coingecko.com/api/v3/simple/price?ids=tether&vs_currencies=vnd';

/**
 * Fetches the latest USDT→VND exchange rate from CoinGecko.
 * Results are cached for 15 minutes in the local database.
 */
export async function getExchangeRate(): Promise<number> {
  const cached = await getCached<{ rate: number }>(CACHE_KEY);
  if (cached) return cached.rate;

  try {
    const response = await fetch(COINGECKO_API, {
      signal: AbortSignal.timeout(5000), // 5s timeout
      cache: 'no-store',
    });
    if (!response.ok) throw new Error(`CoinGecko API returned ${response.status}`);
    const data = await response.json();
    const rate = data.tether?.vnd || 25400;

    await setCache(CACHE_KEY, { rate }, 900); // 15 min cache
    return rate;
  } catch (error) {
    const networkError = isAppError(error)
      ? error
      : new NetworkError('Failed to fetch live USDT/VND rate', {
          context: { source: 'coingecko', endpoint: COINGECKO_API },
        });
    console.warn('[ExchangeRateService]', networkError.message, '- using fallback');
    return 25400; // Static fallback
  }
}
