import { arrayAt, dataOf, dateOf, num, objectAt, str, url, type Json } from './parse';

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

export type FundSort = 'return12m' | 'annualized36m' | 'nav' | 'code';

/** Funds ordered by a column: numbers high to low with unknowns last, codes A-Z. */
export function sortFunds(funds: FundSummary[], key: FundSort): FundSummary[] {
  if (key === 'code') return [...funds].sort((a, b) => a.code.localeCompare(b.code));
  return [...funds].sort((a, b) => (b[key] ?? -Infinity) - (a[key] ?? -Infinity));
}
