import type { BankRate } from '$lib/api/types';

// One bank-rates list (ROADMAP C5): the full list lives on the Fmarket page;
// other pages show the best rate and link to it.

/** Link (relative to the app base) to the full list of bank savings rates. */
export const BANK_RATES_HREF = '/investments/funds?tab=bank';

/** The highest published rate, or null when no bank published one. */
export function bestBankRate(rates: BankRate[]): BankRate | null {
  return rates.reduce<BankRate | null>(
    (best, rate) => (rate.rate > 0 && (!best || rate.rate > best.rate) ? rate : best),
    null,
  );
}
