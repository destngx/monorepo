import type { Currency, Holding } from '$lib/api/types';

/** VND per unit of each foreign currency; a missing rate means unknown. */
export type Rates = Partial<Record<Exclude<Currency, 'VND'>, number>>;

export function toVND(value: number | null, currency: Currency, rates: Rates): number | null {
  if (value === null) return null;
  if (currency === 'VND') return value;
  const rate = rates[currency];
  return rate ? value * rate : null;
}

export interface SectionTotal {
  /** Sum of holdings that could be valued in VND. */
  vnd: number;
  /** Holdings with no value or no rate for their currency. */
  unvalued: number;
  /** Raw sums per original currency. */
  byCurrency: Partial<Record<Currency, number>>;
}

export function sectionTotal(holdings: Holding[], rates: Rates): SectionTotal {
  const total: SectionTotal = { vnd: 0, unvalued: 0, byCurrency: {} };
  for (const holding of holdings) {
    if (holding.value !== null) {
      total.byCurrency[holding.currency] = (total.byCurrency[holding.currency] ?? 0) + holding.value;
    }
    const vnd = toVND(holding.value, holding.currency, rates);
    if (vnd === null) total.unvalued += 1;
    else total.vnd += vnd;
  }
  return total;
}
