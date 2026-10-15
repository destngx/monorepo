import { describe, expect, it } from 'vitest';
import type {
  CardBankReport,
  CardPeriodHistory,
  CardReport,
  CashbackRule,
  PeriodStanding,
  RuleStanding,
} from '$lib/api/types';
import {
  allCards,
  capsLeft,
  cardPeriodCap,
  cardTagTone,
  compareCards,
  cumulative,
  formatMccs,
  historyYears,
  parseCardPayment,
  portfolioTotals,
  visibleCards,
  RECONCILIATION,
} from './cards';

const rule = (cap: number, extra: Partial<CashbackRule> = {}): CashbackRule => ({
  name: 'r',
  mccs: [{ from: 5311, to: 5311 }],
  rate: 1,
  cap,
  minTransaction: 0,
  minCycleSpend: 0,
  ...extra,
});
const amount = (confirmed: number, total: number) => ({ confirmed, estimated: total - confirmed, total });
const month = (m: string, refund: number): CardPeriodHistory => ({
  month: m,
  spent: refund * 100,
  cashback: amount(refund, refund * 2),
  refund,
  status: 'match',
  efficiency: 50,
  missed: refund,
});

describe('cardPeriodCap', () => {
  it('prefers the card-level cap', () => {
    expect(cardPeriodCap({ cap: 300_000, rules: [rule(0)] })).toBe(300_000);
  });

  it('sums rule caps only when every rule is capped', () => {
    expect(cardPeriodCap({ cap: 0, rules: [rule(100), rule(200, { mccs: [{ from: 5411, to: 5411 }] })] })).toBe(300);
    expect(cardPeriodCap({ cap: 0, rules: [rule(100), rule(0, { mccs: [{ from: 5411, to: 5411 }] })] })).toBeNull();
    expect(cardPeriodCap({ cap: 0, rules: [] })).toBeNull();
  });

  it('ignores exclusions and counts only the largest spend tier of a rule', () => {
    const groceries = { mccs: [{ from: 5411, to: 5411 }] };
    expect(
      cardPeriodCap({
        cap: 0,
        rules: [
          rule(100, groceries),
          rule(250, { ...groceries, minCycleSpend: 5_000_000 }),
          rule(0, { mccs: [{ from: 4900, to: 4900 }], exclude: true }),
          rule(50, { mccs: [], fallback: true }),
        ],
      }),
    ).toBe(300);
  });

  it('counts a cap group once', () => {
    const supermarket = rule(680, { mccs: [{ from: 5411, to: 5411 }], capGroup: 'Everyday' });
    const transport = rule(680, { mccs: [{ from: 4121, to: 4121 }], capGroup: 'Everyday' });
    expect(cardPeriodCap({ cap: 0, rules: [supermarket, transport] })).toBe(680);
    expect(cardPeriodCap({ cap: 0, rules: [supermarket, transport, rule(100)] })).toBe(780);
  });
});

describe('capsLeft', () => {
  const standing = (cap: number, capLeft: number, rules: Partial<RuleStanding>[]): PeriodStanding =>
    ({
      cap,
      capLeft,
      rules: rules.map((r) => ({ name: 'r', earned: amount(0, 0), cap: 0, capLeft: 0, ...r })),
    }) as PeriodStanding;

  it('shows the card-level cap when the programme has one', () => {
    expect(capsLeft(standing(300, 120, [{ name: 'Groceries', cap: 100, capLeft: 40 }]))).toEqual([
      { name: 'Card', left: 120, cap: 300 },
    ]);
  });

  it('otherwise shows each capped rule or cap group, skipping uncapped ones', () => {
    const groups = standing(0, 0, [
      { name: 'Supermarket & transport', rules: ['Supermarket', 'Transport'], cap: 680, capLeft: 550 },
      { name: 'Others', cap: 0, capLeft: 0 },
    ]);
    expect(capsLeft(groups)).toEqual([{ name: 'Supermarket & transport', left: 550, cap: 680 }]);
    expect(capsLeft(standing(0, 0, [{ name: 'Others' }]))).toEqual([]);
  });
});

describe('bank totals', () => {
  const bank = (name: string, limit: number, usage: number, cards: string[]): CardBankReport =>
    ({
      name,
      limit,
      usage,
      cashback: amount(10, 30),
      refundYtd: 5,
      feesPaidYtd: 2,
      netEarnYtd: 3,
      cards: cards.map((card) => ({ name: card })),
    }) as CardBankReport;
  const banks = [bank('Acme', 20_000_000, 8_000_000, ['Gold', 'Silver']), bank('Beta', 0, 1_000_000, ['Platinum'])];

  it('lists every held card across banks', () => {
    expect(allCards(banks).map((card) => card.name)).toEqual(['Gold', 'Silver', 'Platinum']);
  });

  it('hides closed cards unless asked to show them', () => {
    const cards = [{ name: 'Gold' }, { name: 'Old', closedFrom: '2026-10-01' }] as CardReport[];
    expect(visibleCards(cards, false).map((card) => card.name)).toEqual(['Gold']);
    expect(visibleCards(cards, true).map((card) => card.name)).toEqual(['Gold', 'Old']);
  });

  it('adds up limits, usage and cashback per bank, not per card', () => {
    expect(portfolioTotals(banks)).toEqual({
      limit: 20_000_000,
      usage: 9_000_000,
      cashback: amount(20, 60),
      refundYtd: 10,
      feesPaidYtd: 4,
      netEarnYtd: 6,
    });
  });
});

describe('formatMccs', () => {
  it('writes codes and ranges as four digits', () => {
    expect(
      formatMccs([
        { from: 742, to: 742 },
        { from: 5812, to: 5814 },
      ]),
    ).toBe('0742, 5812-5814');
    expect(formatMccs(undefined)).toBe('');
  });
});

describe('cumulative', () => {
  it('returns the running total', () => {
    expect(cumulative([1, 2, 3])).toEqual([1, 3, 6]);
    expect(cumulative([])).toEqual([]);
  });
});

describe('reconciliation labels', () => {
  it('flags only the statuses that need the user', () => {
    expect(RECONCILIATION.below.tone).toBe('red');
    expect(RECONCILIATION.above.tone).toBe('amber');
    expect(RECONCILIATION.match.tone).toBe('green');
    expect(RECONCILIATION.open.tone).toBe('neutral');
  });
});

describe('cross-card comparison', () => {
  const cards = [
    { name: 'Visa', history: [month('2026-02', 20), month('2026-01', 10), month('2025-12', 5)] },
    { name: 'Master', history: [month('2026-03', 7)] },
  ];

  it('compares the cashback total of each period', () => {
    expect(compareCards(cards, '2025', 'cashback').series[0].values).toEqual([10]);
  });

  it('lists years newest first', () => {
    expect(historyYears(cards)).toEqual(['2026', '2025']);
  });

  it('aligns every card on the union of months with gaps as null', () => {
    expect(compareCards(cards, '2026', 'refund')).toEqual({
      months: ['2026-01', '2026-02', '2026-03'],
      series: [
        { name: 'Visa', values: [10, 20, null] },
        { name: 'Master', values: [null, null, 7] },
      ],
    });
  });
});

describe('cardTagTone', () => {
  const keywords = ['silver', 'Gold', 'rose gold'];

  it('gives each card a stable tone by keyword, whatever the order the cards come in', () => {
    expect(cardTagTone('gold online', keywords)).toBe('green');
    expect(cardTagTone('Silver Others', keywords)).toBe('violet');
    expect(cardTagTone('gold online', [...keywords].reverse())).toBe('green');
  });

  it('prefers the longest keyword a tag contains', () => {
    expect(cardTagTone('rose gold travel', keywords)).toBe('blue');
  });

  it('leaves tags of no card neutral', () => {
    expect(cardTagTone('groceries', keywords)).toBe('neutral');
    expect(cardTagTone('gold', [])).toBe('neutral');
    expect(cardTagTone('gold', [' '])).toBe('neutral');
  });
});

describe('parseCardPayment', () => {
  it('reads the payee, a grouped amount and an optional MCC', () => {
    expect(parseCardPayment({ payee: ' Coop ', amount: '1,500,000', mcc: '' })).toEqual({
      query: { payee: 'Coop', amount: 1_500_000 },
    });
    expect(parseCardPayment({ payee: '', amount: '200000', mcc: '0742' })).toEqual({
      query: { payee: '', amount: 200_000, mcc: 742 },
    });
  });

  it('explains what is missing or malformed', () => {
    expect(parseCardPayment({ payee: 'Coop', amount: '', mcc: '' })).toEqual({ error: 'Enter a positive amount.' });
    expect(parseCardPayment({ payee: 'Coop', amount: '-5', mcc: '' })).toEqual({ error: 'Enter a positive amount.' });
    expect(parseCardPayment({ payee: 'Coop', amount: '5', mcc: '54x1' })).toEqual({
      error: 'An MCC is 4 digits, e.g. 5411.',
    });
    expect(parseCardPayment({ payee: ' ', amount: '5', mcc: '' })).toEqual({ error: 'Enter the payee or its MCC.' });
  });
});
