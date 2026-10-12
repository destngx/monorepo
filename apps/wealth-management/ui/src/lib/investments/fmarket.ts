// Fmarket explorer (WM-INV-10). The engine proxies fmarket's own JSON
// untouched, so everything here parses defensively: a missing or oddly typed
// field becomes null instead of breaking a whole section.

type Json = Record<string, unknown>;

const isObject = (value: unknown): value is Json =>
  typeof value === 'object' && value !== null && !Array.isArray(value);
const objectAt = (value: unknown): Json => (isObject(value) ? value : {});
const arrayAt = (value: unknown): unknown[] => (Array.isArray(value) ? value : []);

function num(value: unknown): number | null {
  if (typeof value === 'number') return Number.isFinite(value) ? value : null;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

function str(value: unknown): string | null {
  if (typeof value === 'string') return value.trim() || null;
  if (typeof value === 'number' && Number.isFinite(value)) return String(value);
  return null;
}

/** An absolute http(s) URL, or null; keeps odd values out of href and src. */
function url(value: unknown): string | null {
  const text = str(value);
  if (!text) return null;
  try {
    const parsed = new URL(text);
    return parsed.protocol === 'https:' || parsed.protocol === 'http:' ? parsed.href : null;
  } catch {
    return null;
  }
}

/** fmarket wraps payloads as `{status, code, message, data, extra}`. */
const dataOf = (payload: unknown): unknown => objectAt(payload).data;

/** Epoch milliseconds as a calendar date in Vietnam (UTC+7, no DST). */
export function vnDate(ms: number): string {
  return new Date(ms + 7 * 3_600_000).toISOString().slice(0, 10);
}

function dateOf(value: unknown): string | null {
  const ms = num(value);
  if (ms !== null) return vnDate(ms);
  const text = str(value);
  return text && /^\d{4}-\d{2}-\d{2}/.test(text) ? text.slice(0, 10) : null;
}

// Fund lists

export type FundType = 'STOCK' | 'BOND' | 'BALANCED' | 'MMF';

export const FUND_TYPES: { key: FundType; label: string }[] = [
  { key: 'STOCK', label: 'Stock' },
  { key: 'BOND', label: 'Bond' },
  { key: 'BALANCED', label: 'Balanced' },
  { key: 'MMF', label: 'Money market' },
];

/** getProductsFilterNav params for one fund type; MMF is a flag, not an asset type. */
export function fundListParams(type: FundType): Record<string, unknown> {
  return type === 'MMF'
    ? { page: 1, pageSize: 100, assetTypes: [], isMMFFund: true }
    : { page: 1, pageSize: 100, assetTypes: [type], isMMFFund: false };
}

export interface FundSummary {
  id: number;
  /** The short name, which is also what fmarket's product page is keyed by. */
  code: string;
  name: string;
  issuer: string | null;
  issuerLogo: string | null;
  assetType: string | null;
  /** fmarket's asset type code: STOCK, BOND, BALANCED. */
  assetCode: string | null;
  nav: number | null;
  /** Percent change over 12 months. */
  return12m: number | null;
  /** Annualised percent return over 36 months. */
  annualized36m: number | null;
  updatedAt: string | null;
}

function parseFund(raw: unknown): FundSummary | null {
  const row = objectAt(raw);
  const id = num(row.id);
  const code = str(row.shortName) ?? str(row.code);
  if (id === null || !code) return null;
  const owner = objectAt(row.owner);
  const change = objectAt(row.productNavChange);
  return {
    id,
    code,
    name: str(row.name) ?? code,
    issuer: str(owner.shortName) ?? str(owner.name) ?? (typeof row.owner === 'string' ? row.owner : null),
    issuerLogo: url(owner.avatarUrl),
    assetType: str(objectAt(row.dataFundAssetType).name),
    assetCode: str(objectAt(row.dataFundAssetType).code),
    nav: num(row.nav),
    return12m: num(change.navTo12Months),
    annualized36m: num(change.annualizedReturn36Months),
    updatedAt: dateOf(change.updateAt),
  };
}

/** Rows of a getProductsFilterNav response, best 12-month return first, unknown returns last. */
export function parseFundList(payload: unknown): FundSummary[] {
  const data = objectAt(dataOf(payload));
  const rows = arrayAt(data.rows ?? data.list);
  return rows
    .map(parseFund)
    .filter((fund): fund is FundSummary => fund !== null)
    .sort((a, b) => (b.return12m ?? -Infinity) - (a.return12m ?? -Infinity));
}

// Fund detail

export interface FundFee {
  type: string;
  program: string | null;
  condition: string;
  /** Percent of the transaction value. */
  fee: number;
  /** Target funds of a switch fee. */
  targets: string[];
}

export interface FundHolding {
  code: string;
  industry: string | null;
  percent: number;
}

export interface FundShare {
  label: string;
  percent: number;
  color?: string;
}

export interface FundDetail extends FundSummary {
  description: string | null;
  riskLevel: string | null;
  inceptionDate: string | null;
  managementFee: number | null;
  performanceFee: number | null;
  buyMinValue: number | null;
  sellMinUnits: number | null;
  holdingMinUnits: number | null;
  orderCutoff: string | null;
  nextTradingDate: string | null;
  website: string | null;
  holdingsAsOf: string | null;
  /** Percent change by fmarket period key (navTo1Months, navToLastYear, ...). */
  changes: Record<string, number | null>;
  fees: FundFee[];
  topHoldings: FundHolding[];
  allocation: FundShare[];
  industries: FundShare[];
}

const operatorWords: Record<string, string> = { '<': 'under', '<=': 'up to', '>': 'above', '>=': 'from' };

/** A fee tier's bounds in words, e.g. "Under 12 months" or "12-24 months". */
export function feeCondition(raw: unknown): string {
  const row = objectAt(raw);
  const isMonths = str(row.feeUnitType) === 'MONTH';
  const unit = (value: number) =>
    isMonths ? `${value} month${value === 1 ? '' : 's'}` : `${new Intl.NumberFormat('vi-VN').format(value)} ₫`;
  const begin = num(row.beginVolume);
  const end = num(row.endVolume);
  const beginOp = str(objectAt(row.beginRelationalOperator).code) ?? '>=';
  const endOp = str(objectAt(row.endRelationalOperator).code) ?? '<';
  const capitalise = (text: string) => text.charAt(0).toUpperCase() + text.slice(1);
  if (end !== null && (begin === null || begin === 0))
    return capitalise(`${operatorWords[endOp] ?? endOp} ${unit(end)}`);
  if (end === null && begin !== null && begin > 0)
    return capitalise(`${operatorWords[beginOp] ?? beginOp} ${unit(begin)}`);
  if (end !== null && begin !== null) return isMonths ? `${begin}-${end} months` : `${unit(begin)} - ${unit(end)}`;
  return 'Any amount';
}

/**
 * Fee tiers in fmarket order. Switch fees repeat the same schedule per target
 * fund, so identical tiers are merged and list their targets instead.
 */
export function parseFees(list: unknown): FundFee[] {
  const fees: FundFee[] = [];
  for (const raw of arrayAt(list)) {
    const row = objectAt(raw);
    const fee = num(row.fee);
    const type = str(row.type);
    if (fee === null || !type) continue;
    const program = str(objectAt(row.productProgram).name);
    const condition = feeCondition(row);
    const target = str(objectAt(row.destProduct).shortName);
    const same = fees.find(
      (f) => f.type === type && f.program === program && f.condition === condition && f.fee === fee,
    );
    if (same) {
      if (target && !same.targets.includes(target)) same.targets.push(target);
      continue;
    }
    fees.push({ type, program, condition, fee, targets: target ? [target] : [] });
  }
  return fees;
}

function parseShares(list: unknown, read: (row: Json) => FundShare | null): FundShare[] {
  return arrayAt(list)
    .map((raw) => read(objectAt(raw)))
    .filter((share): share is FundShare => share !== null && share.percent > 0)
    .sort((a, b) => b.percent - a.percent);
}

/** A getProductDetails response, or null when it carries no product. */
export function parseFundDetail(payload: unknown): FundDetail | null {
  const data = objectAt(dataOf(payload));
  const summary = parseFund(data);
  if (!summary) return null;
  const change = objectAt(data.productNavChange);
  const changes: Record<string, number | null> = {};
  for (const [key, value] of Object.entries(change)) if (key.startsWith('navTo')) changes[key] = num(value);
  const session = objectAt(data.productTradingSession);
  const holdings = [...arrayAt(data.productTopHoldingList), ...arrayAt(data.productTopHoldingBondList)]
    .map((raw) => {
      const row = objectAt(raw);
      const code = str(row.stockCode) ?? str(row.bondCode) ?? str(row.code);
      const percent = num(row.netAssetPercent);
      return code && percent !== null ? { code, industry: str(row.industry), percent } : null;
    })
    .filter((holding): holding is FundHolding => holding !== null)
    .sort((a, b) => b.percent - a.percent)
    .slice(0, 10);
  return {
    ...summary,
    description: str(data.description),
    riskLevel: str(objectAt(data.riskLevel).name),
    inceptionDate: dateOf(data.firstIssueAt),
    managementFee: num(data.managementFee),
    performanceFee: num(data.performanceFee),
    buyMinValue: num(data.buyMinValue),
    sellMinUnits: num(data.sellMin),
    holdingMinUnits: num(data.holdingMin),
    orderCutoff: str(session.closedOrderBookTimeString),
    nextTradingDate: str(session.tradingTimeString),
    website: url(data.websiteURL) ?? url(data.website),
    holdingsAsOf: str(objectAt(data.productFund).updateAssetHoldingTime),
    changes,
    fees: parseFees(data.productFeeList),
    topHoldings: holdings,
    allocation: parseShares(data.productAssetHoldingList, (row) => {
      const type = objectAt(row.assetType);
      const label = str(type.name);
      const percent = num(row.assetPercent);
      return label && percent !== null ? { label, percent, color: str(type.colorCode) ?? undefined } : null;
    }),
    industries: parseShares(data.productIndustriesHoldingList, (row) => {
      const label = str(row.industry);
      const percent = num(row.assetPercent);
      return label && percent !== null ? { label, percent } : null;
    }),
  };
}

// Time series and ranges

export interface Point {
  date: string;
  value: number;
}

export type Range = '1M' | '3M' | '6M' | 'YTD' | '1Y' | '3Y' | '5Y' | 'ALL';
export const RANGES: Range[] = ['1M', '3M', '6M', 'YTD', '1Y', '3Y', '5Y', 'ALL'];
export type RangeHandler = (range: Range) => void;
export type ValueFormat = (value: number) => string;

const rangeMonths: Partial<Record<Range, number>> = { '1M': 1, '3M': 3, '6M': 6, '1Y': 12, '3Y': 36, '5Y': 60 };

/** First calendar day ("YYYY-MM-DD") a range covers up to `today`; null for ALL. */
export function rangeStart(range: Range, today: string): string | null {
  const [year, month, day] = today.split('-').map(Number);
  if (range === 'ALL') return null;
  if (range === 'YTD') return `${year}-01-01`;
  const months = rangeMonths[range] ?? 12;
  // Clamp the day so 31 March minus one month is 28/29 February, not 3 March.
  const target = new Date(Date.UTC(year, month - 1 - months, 1));
  const lastDay = new Date(Date.UTC(target.getUTCFullYear(), target.getUTCMonth() + 1, 0)).getUTCDate();
  target.setUTCDate(Math.min(day, lastDay));
  return target.toISOString().slice(0, 10);
}

/** Earliest date asked for when a history range is ALL; fmarket gold starts in 2000. */
export const HISTORY_ORIGIN = '2000-01-01';

/** fromDate/toDate params (yyyyMMdd) for the gold and USD history actions. */
export function historyParams(range: Range, today: string): { fromDate: string; toDate: string } {
  const compact = (date: string) => date.replaceAll('-', '');
  return { fromDate: compact(rangeStart(range, today) ?? HISTORY_ORIGIN), toDate: compact(today) };
}

/** Points on or after the range start; points must be oldest first. */
export function sliceRange<T extends { date: string }>(points: T[], range: Range, today: string): T[] {
  const start = rangeStart(range, today);
  return start ? points.filter((point) => point.date >= start) : points;
}

/** Sorts oldest first and keeps the last row of each day. */
function ascendingByDate<T extends { date: string }>(rows: T[]): T[] {
  const byDate = new Map<string, T>();
  for (const row of [...rows].sort((a, b) => a.date.localeCompare(b.date))) byDate.set(row.date, row);
  return [...byDate.values()];
}

/** getNavHistory rows ({nav, navDate}) as NAV points, oldest first. */
export function parseNavHistory(payload: unknown): Point[] {
  const points = arrayAt(dataOf(payload)).flatMap((raw) => {
    const row = objectAt(raw);
    const date = dateOf(row.navDate) ?? dateOf(row.createdAt);
    const value = num(row.nav);
    return date && value !== null && value > 0 ? [{ date, value }] : [];
  });
  return ascendingByDate(points);
}

/** Percent change from the first to the last point. */
export function periodReturn(points: Point[]): number | null {
  if (points.length < 2 || points[0].value === 0) return null;
  return (points[points.length - 1].value / points[0].value - 1) * 100;
}

/** Keeps about `max` evenly spaced points, always including the first and last. */
export function downsample<T>(points: T[], max = 400): T[] {
  if (points.length <= max || max < 2) return points;
  const step = (points.length - 1) / (max - 1);
  return Array.from({ length: max }, (_, index) => points[Math.round(index * step)]);
}

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

export type FundSort = 'return12m' | 'annualized36m' | 'nav' | 'code';

/** Funds ordered by a column: numbers high to low with unknowns last, codes A-Z. */
export function sortFunds(funds: FundSummary[], key: FundSort): FundSummary[] {
  if (key === 'code') return [...funds].sort((a, b) => a.code.localeCompare(b.code));
  return [...funds].sort((a, b) => (b[key] ?? -Infinity) - (a[key] ?? -Infinity));
}

// Sections load independently, so one failing fmarket action never blanks the page.

export type Section<T> = { ok: true; value: T } | { ok: false; error: string };

export async function settle<T>(load: () => Promise<T>): Promise<Section<T>> {
  try {
    return { ok: true, value: await load() };
  } catch (cause) {
    return { ok: false, error: cause instanceof Error ? cause.message : String(cause) };
  }
}
