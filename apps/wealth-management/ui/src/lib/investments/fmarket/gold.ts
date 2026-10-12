import { arrayAt, dataOf, dateOf, num, objectAt } from './parse';
import { ascendingByDate, type Point } from './series';

// Gold and USD

export interface GoldRow {
  date: string;
  /** World gold, USD per troy ounce. */
  world: number | null;
  /** SJC bar, VND per lượng: what dealers pay (bid) and charge (ask). */
  sjcBid: number | null;
  sjcAsk: number | null;
}

export interface GoldSnapshot {
  usdRate: number | null;
  /** 9999 ring gold, VND per lượng. */
  ringBid: number | null;
  ringAsk: number | null;
  /** fmarket's own world price conversion, VND per lượng. */
  worldVnd: number | null;
  goldChangePercent: number | null;
  sjcChangePercent: number | null;
}

/** getGoldPriceHistory rows ({price, bidSjc, askSjc, reportDate}); fmarket sends them newest first. */
export function parseGoldHistory(payload: unknown): GoldRow[] {
  const rows = arrayAt(dataOf(payload)).flatMap((raw) => {
    const row = objectAt(raw);
    const date = dateOf(row.reportDate);
    if (!date) return [];
    const positive = (value: unknown) => {
      const parsed = num(value);
      return parsed !== null && parsed > 0 ? parsed : null;
    };
    return [{ date, world: positive(row.price), sjcBid: positive(row.bidSjc), sjcAsk: positive(row.askSjc) }];
  });
  return ascendingByDate(rows);
}

/**
 * The `extra` block of getGoldPriceHistory. fmarket names ring prices from
 * the buyer's side: goldRingPriceBuy is what a buyer pays (the dealer's ask).
 */
export function parseGoldSnapshot(payload: unknown): GoldSnapshot {
  const extra = objectAt(objectAt(payload).extra);
  return {
    usdRate: num(extra.rateUsdToVnd),
    ringBid: num(extra.goldRingPriceSell),
    ringAsk: num(extra.goldRingPriceBuy) ?? num(extra.goldRingPrice),
    worldVnd: num(extra.ratePriceGoldWorldToVND),
    goldChangePercent: num(extra.goldPriceChangePercent),
    sjcChangePercent: num(extra.goldSjcChangePercent),
  };
}

/** getUsdRateHistory rows ({rateSellUsd, reportDate}) as USD/VND points, oldest first. */
export function parseUsdHistory(payload: unknown): Point[] {
  const points = arrayAt(dataOf(payload)).flatMap((raw) => {
    const row = objectAt(raw);
    const date = dateOf(row.reportDate);
    const value = num(row.rateSellUsd);
    return date && value !== null && value > 0 ? [{ date, value }] : [];
  });
  return ascendingByDate(points);
}

/** Troy ounces per lượng (37.5 g / 31.1035 g). */
export const OUNCES_PER_LUONG = 1.2056;

/** World gold in million VND per lượng: price × 1.2056 × usdRate / 1e6. */
export function worldGoldMillionVND(priceUsdPerOunce: number, usdRate: number): number {
  return (priceUsdPerOunce * OUNCES_PER_LUONG * usdRate) / 1e6;
}

export interface GoldChartRow {
  date: string;
  sjcBid: number | null;
  sjcAsk: number | null;
  world: number | null;
}

/**
 * Gold rows in million VND per lượng. World gold converts at the USD rate of
 * the same day (or the latest before it), falling back to `fallbackRate`
 * before the USD history starts.
 */
export function goldChartRows(gold: GoldRow[], usd: Point[], fallbackRate: number | null): GoldChartRow[] {
  let index = -1;
  return gold.map((row) => {
    while (index + 1 < usd.length && usd[index + 1].date <= row.date) index++;
    const rate = index >= 0 ? usd[index].value : fallbackRate;
    const million = (value: number | null) => (value === null ? null : value / 1e6);
    return {
      date: row.date,
      sjcBid: million(row.sjcBid),
      sjcAsk: million(row.sjcAsk),
      world: row.world !== null && rate ? worldGoldMillionVND(row.world, rate) : null,
    };
  });
}

/** Latest non-null value of a field, scanning newest first. */
export function latest<T, K extends keyof T>(rows: T[], key: K): NonNullable<T[K]> | null {
  for (let index = rows.length - 1; index >= 0; index--) {
    const value = rows[index][key];
    if (value !== null && value !== undefined) return value;
  }
  return null;
}
