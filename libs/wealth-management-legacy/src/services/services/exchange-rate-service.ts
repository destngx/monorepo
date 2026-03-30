import { NetworkError, isAppError } from '../../utils/errors';

const COINGECKO_API = 'https://api.coingecko.com/api/v3/simple/price?ids=tether&vs_currencies=vnd';

/**
 * Fetches the latest USDT→VND exchange rate from CoinGecko.
 */
export async function getExchangeRate(): Promise<number> {
  try {
    const response = await fetch(COINGECKO_API, {
      signal: AbortSignal.timeout(5000), // 5s timeout
      cache: 'no-store',
    });
    if (!response.ok) throw new Error(`CoinGecko API returned ${response.status}`);
    const data = await response.json();
    return data.tether?.vnd || 25400;
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
