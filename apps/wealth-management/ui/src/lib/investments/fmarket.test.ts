import { describe, expect, it } from 'vitest';
import {
  downsample,
  feeCondition,
  fundListParams,
  goldChartRows,
  historyParams,
  latest,
  settle,
  sortFunds,
  parseFees,
  parseFundDetail,
  parseFundList,
  parseGoldHistory,
  parseGoldSnapshot,
  parseNavHistory,
  parseUsdHistory,
  periodReturn,
  rangeStart,
  sliceRange,
  vnDate,
  worldGoldMillionVND,
} from './fmarket';

// Epoch ms of midnight in Vietnam, as fmarket reports dates.
const vnMidnight = (date: string) => Date.parse(`${date}T00:00:00+07:00`);

describe('vnDate', () => {
  it('reads epoch ms as the Vietnam calendar day', () => {
    expect(vnDate(1791738000000)).toBe('2026-10-12');
  });
});

describe('fund lists', () => {
  const payload = {
    status: 200,
    data: {
      total: 3,
      rows: [
        {
          id: 1,
          shortName: 'AAA',
          name: 'Fund A',
          nav: 11280.75,
          owner: { shortName: 'SGIC', avatarUrl: 'https://x/a.png' },
          dataFundAssetType: { code: 'STOCK', name: 'Quỹ cổ phiếu' },
          productNavChange: { navTo12Months: 5.27, annualizedReturn36Months: 12.5, updateAt: vnMidnight('2026-10-12') },
        },
        { id: 2, shortName: 'BBB', nav: '9000', productNavChange: { navTo12Months: null } },
        { id: 3, shortName: 'CCC', productNavChange: { navTo12Months: 9.1 } },
        { shortName: 'no id' },
        'garbage',
      ],
    },
  };

  it('parses rows, skips invalid ones and ranks by 12-month return', () => {
    const funds = parseFundList(payload);
    expect(funds.map((f) => f.code)).toEqual(['CCC', 'AAA', 'BBB']);
    expect(funds[1]).toEqual({
      id: 1,
      code: 'AAA',
      name: 'Fund A',
      issuer: 'SGIC',
      issuerLogo: 'https://x/a.png',
      assetType: 'Quỹ cổ phiếu',
      assetCode: 'STOCK',
      nav: 11280.75,
      return12m: 5.27,
      annualized36m: 12.5,
      updatedAt: '2026-10-12',
    });
    expect(funds[2].nav).toBe(9000);
    expect(funds[2].name).toBe('BBB');
  });

  it('survives malformed payloads', () => {
    expect(parseFundList(null)).toEqual([]);
    expect(parseFundList({ data: { rows: 'x' } })).toEqual([]);
    expect(parseFundList({ data: { list: [{ id: 9, code: 'L' }] } })[0].code).toBe('L');
  });

  it('asks for money market funds by flag', () => {
    expect(fundListParams('MMF')).toMatchObject({ isMMFFund: true, assetTypes: [] });
    expect(fundListParams('BOND')).toMatchObject({ isMMFFund: false, assetTypes: ['BOND'] });
  });
});

describe('fees', () => {
  const tier = (type: string, begin: number, end: number | null, fee: number, extra: object = {}) => ({
    type,
    beginVolume: begin,
    endVolume: end,
    beginRelationalOperator: { code: begin === 0 ? '>' : '>=' },
    endRelationalOperator: { code: '<' },
    fee,
    feeUnitType: 'MONTH',
    productProgram: { name: 'Linh hoạt' },
    ...extra,
  });

  it('describes tier bounds in words', () => {
    expect(feeCondition(tier('SELL', 0, 12, 1.5))).toBe('Under 12 months');
    expect(feeCondition(tier('SELL', 12, 24, 0.5))).toBe('12-24 months');
    expect(feeCondition(tier('SELL', 24, null, 0))).toBe('From 24 months');
    expect(
      feeCondition({ ...tier('BUY', 0, 100_000_000, 0), feeUnitType: 'MONEY', endRelationalOperator: { code: '<=' } }),
    ).toBe('Up to 100.000.000 ₫');
    expect(feeCondition({})).toBe('Any amount');
  });

  it('merges switch tiers that only differ by target fund', () => {
    const fees = parseFees([
      tier('SELL', 0, 12, 1.5),
      tier('TRANSFER', 0, 12, 1.5, { destProduct: { shortName: 'DCDE' } }),
      tier('TRANSFER', 0, 12, 1.5, { destProduct: { shortName: 'DCBF' } }),
      tier('TRANSFER', 12, null, 0, { destProduct: { shortName: 'DCDE' } }),
      { type: 'BUY' },
    ]);
    expect(fees).toHaveLength(3);
    expect(fees[1]).toEqual({
      type: 'TRANSFER',
      program: 'Linh hoạt',
      condition: 'Under 12 months',
      fee: 1.5,
      targets: ['DCDE', 'DCBF'],
    });
  });
});

describe('parseFundDetail', () => {
  it('reads overview, minimums, top-10 holdings and allocation', () => {
    const holdings = Array.from({ length: 12 }, (_, i) => ({
      stockCode: `S${i}`,
      industry: 'Bank',
      netAssetPercent: i + 1,
    }));
    const detail = parseFundDetail({
      data: {
        id: 28,
        code: 'VFMVF1',
        shortName: 'DCDS',
        name: 'Dragon',
        nav: 93195.8,
        buyMinValue: 100000,
        sellMin: 10,
        holdingMin: 10,
        managementFee: 1.95,
        firstIssueAt: vnMidnight('2004-05-20'),
        riskLevel: { name: 'Trung bình' },
        productNavChange: { navTo1Months: -2.54, navToBeginning: 831.96, updateAt: 1 },
        productTopHoldingList: holdings,
        productAssetHoldingList: [
          { assetPercent: 2.16, assetType: { name: 'Bond', colorCode: '#FA9F05' } },
          { assetPercent: 82.33, assetType: { name: 'Stock', colorCode: '#3FEBA2' } },
          { assetPercent: 0, assetType: { name: 'Empty' } },
        ],
        productIndustriesHoldingList: [{ industry: 'Bank', assetPercent: 32.55 }],
        productTradingSession: { closedOrderBookTimeString: '14:30 13/10/2026', tradingTimeString: '14/10/2026' },
        productFund: { updateAssetHoldingTime: '09/2026' },
      },
    });
    expect(detail).not.toBeNull();
    expect(detail?.code).toBe('DCDS');
    expect(detail?.inceptionDate).toBe('2004-05-20');
    expect(detail?.changes).toEqual({ navTo1Months: -2.54, navToBeginning: 831.96 });
    expect(detail?.topHoldings).toHaveLength(10);
    expect(detail?.topHoldings[0]).toEqual({ code: 'S11', industry: 'Bank', percent: 12 });
    expect(detail?.allocation.map((a) => a.label)).toEqual(['Stock', 'Bond']);
    expect(detail?.allocation[0].color).toBe('#3FEBA2');
    expect(detail?.orderCutoff).toBe('14:30 13/10/2026');
    expect(detail?.holdingsAsOf).toBe('09/2026');
  });

  it('is null without a product', () => {
    expect(parseFundDetail({ data: null })).toBeNull();
    expect(parseFundDetail('x')).toBeNull();
  });
});

describe('ranges', () => {
  it('starts each range on the right day', () => {
    expect(rangeStart('1M', '2026-10-13')).toBe('2026-09-13');
    expect(rangeStart('1M', '2026-03-31')).toBe('2026-02-28');
    expect(rangeStart('YTD', '2026-10-13')).toBe('2026-01-01');
    expect(rangeStart('3Y', '2026-10-13')).toBe('2023-10-13');
    expect(rangeStart('ALL', '2026-10-13')).toBeNull();
  });

  it('builds history params in yyyyMMdd', () => {
    expect(historyParams('1Y', '2026-10-13')).toEqual({ fromDate: '20251013', toDate: '20261013' });
    expect(historyParams('ALL', '2026-10-13')).toEqual({ fromDate: '20000101', toDate: '20261013' });
  });

  it('slices points to the range', () => {
    const points = [
      { date: '2025-01-01', value: 1 },
      { date: '2026-09-20', value: 2 },
      { date: '2026-10-12', value: 3 },
    ];
    expect(sliceRange(points, '1M', '2026-10-13').map((p) => p.value)).toEqual([2, 3]);
    expect(sliceRange(points, 'ALL', '2026-10-13')).toHaveLength(3);
  });
});

describe('series', () => {
  it('sorts NAV history oldest first and drops bad rows', () => {
    const points = parseNavHistory({
      data: [
        { nav: 12, navDate: '2026-10-12' },
        { nav: 10, navDate: '2026-10-10' },
        { nav: 0, navDate: '2026-10-11' },
        { nav: 'x', navDate: '2026-10-09' },
      ],
    });
    expect(points).toEqual([
      { date: '2026-10-10', value: 10 },
      { date: '2026-10-12', value: 12 },
    ]);
    expect(periodReturn(points)).toBeCloseTo(20);
    expect(periodReturn(points.slice(0, 1))).toBeNull();
  });

  it('reverses newest-first gold and USD history', () => {
    const gold = parseGoldHistory({
      data: [
        { price: 4162.96, bidSjc: 141300000, askSjc: 144300000, reportDate: vnMidnight('2026-10-12') },
        { price: 4170.93, bidSjc: 141800000, askSjc: 144800000, reportDate: vnMidnight('2026-10-09') },
        { price: 289, bidSjc: null, askSjc: null, reportDate: vnMidnight('2000-01-03') },
      ],
    });
    expect(gold.map((row) => row.date)).toEqual(['2000-01-03', '2026-10-09', '2026-10-12']);
    expect(gold[0]).toEqual({ date: '2000-01-03', world: 289, sjcBid: null, sjcAsk: null });

    const usd = parseUsdHistory({
      data: [
        { rateSellUsd: 26120, reportDate: vnMidnight('2026-10-12') },
        { rateSellUsd: 26070, reportDate: vnMidnight('2026-10-09') },
      ],
    });
    expect(usd).toEqual([
      { date: '2026-10-09', value: 26070 },
      { date: '2026-10-12', value: 26120 },
    ]);
  });

  it('downsamples evenly and keeps both ends', () => {
    const values = Array.from({ length: 1000 }, (_, i) => i);
    const sampled = downsample(values, 5);
    expect(sampled).toEqual([0, 250, 500, 749, 999]);
    expect(downsample([1, 2], 5)).toEqual([1, 2]);
  });

  it('finds the latest known value', () => {
    expect(latest([{ v: 1 }, { v: 2 }, { v: null }], 'v')).toBe(2);
    expect(latest([{ v: null }], 'v')).toBeNull();
  });
});

describe('gold conversion', () => {
  it('converts USD per ounce to million VND per lượng', () => {
    // 4162.96 × 1.2056 × 26120 / 1e6
    expect(worldGoldMillionVND(4162.96, 26120)).toBeCloseTo(131.0927, 3);
  });

  it('converts each day at that day’s USD rate, falling back before USD history starts', () => {
    const rows = goldChartRows(
      [
        { date: '2019-01-02', world: 1000, sjcBid: 36_000_000, sjcAsk: 36_200_000 },
        { date: '2026-10-09', world: 2000, sjcBid: null, sjcAsk: 144_800_000 },
        { date: '2026-10-12', world: 2000, sjcBid: 141_300_000, sjcAsk: 144_300_000 },
      ],
      [
        { date: '2026-10-08', value: 25000 },
        { date: '2026-10-12', value: 26000 },
      ],
      23000,
    );
    expect(rows[0]).toEqual({
      date: '2019-01-02',
      sjcBid: 36,
      sjcAsk: 36.2,
      world: worldGoldMillionVND(1000, 23000),
    });
    expect(rows[1].world).toBeCloseTo(worldGoldMillionVND(2000, 25000));
    expect(rows[1].sjcBid).toBeNull();
    expect(rows[2].world).toBeCloseTo(worldGoldMillionVND(2000, 26000));
  });

  it('reads current SJC, 9999 ring and USD figures from extra', () => {
    expect(
      parseGoldSnapshot({
        extra: {
          goldRingPriceBuy: 143500000,
          goldRingPriceSell: 139500000,
          rateUsdToVnd: 26120,
          ratePriceGoldWorldToVND: 131098512,
          goldPriceChangePercent: -4.54,
          goldSjcChangePercent: -2.96,
        },
      }),
    ).toEqual({
      usdRate: 26120,
      ringBid: 139500000,
      ringAsk: 143500000,
      worldVnd: 131098512,
      goldChangePercent: -4.54,
      sjcChangePercent: -2.96,
    });
    expect(parseGoldSnapshot(null).usdRate).toBeNull();
  });
});

describe('sortFunds', () => {
  const fund = (code: string, return12m: number | null, nav: number | null) =>
    ({ id: 1, code, name: code, return12m, annualized36m: null, nav }) as Parameters<typeof sortFunds>[0][number];

  it('orders numbers high to low with unknowns last', () => {
    const funds = [fund('B', null, 2), fund('A', 3, null), fund('C', 5, 1)];
    expect(sortFunds(funds, 'return12m').map((f) => f.code)).toEqual(['C', 'A', 'B']);
    expect(sortFunds(funds, 'nav').map((f) => f.code)).toEqual(['B', 'C', 'A']);
    expect(sortFunds(funds, 'code').map((f) => f.code)).toEqual(['A', 'B', 'C']);
  });
});

describe('settle', () => {
  it('captures failures as a section error', async () => {
    expect(await settle(() => Promise.resolve(1))).toEqual({ ok: true, value: 1 });
    expect(await settle(() => Promise.reject(new Error('fmarket request failed with status 400')))).toEqual({
      ok: false,
      error: 'fmarket request failed with status 400',
    });
  });
});

describe('urls', () => {
  it('keeps only http(s) links', () => {
    const detail = parseFundDetail({
      data: { id: 1, shortName: 'X', website: 'javascript:alert(1)', owner: { avatarUrl: 'https://x/a.png' } },
    });
    expect(detail?.website).toBeNull();
    expect(detail?.issuerLogo).toBe('https://x/a.png');
  });
});
