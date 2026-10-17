import { describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { maskMoney, maskSegments, maskText } from './moneyMask';

describe('maskMoney', () => {
  it.each([
    ['Spent 1.500.000 ₫ on food', `Spent ${MASK} on food`],
    ['Spent 1.500.000₫ on food', `Spent ${MASK} on food`],
    ['Spent 1,500,000 VND', `Spent ${MASK}`],
    ['Spent VND 1,500,000 total', `Spent ${MASK} total`],
    ['Paid 250.000đ', `Paid ${MASK}`],
    ['Paid 250.000 đồng', `Paid ${MASK}`],
    ['about 300k per week', `about ${MASK} per week`],
    ['about 300K per week', `about ${MASK} per week`],
    ['income of 25M this month', `income of ${MASK} this month`],
    ['income of 2,5 tr', `income of ${MASK}`],
    ['saved 1,2 tỷ so far', `saved ${MASK} so far`],
    ['saved 500 triệu', `saved ${MASK}`],
    ['BTC at $64,250.12 today', `BTC at ${MASK} today`],
    ['holding 1,234.56 USD', `holding ${MASK}`],
    ['holding 120 USDT', `holding ${MASK}`],
    ['net 1.5M VND', `net ${MASK}`],
    ['overspent by -200.000 ₫', `overspent by ${MASK}`],
    ['**12.000.000 ₫** left', `**${MASK}** left`],
    ['grouped 12.345.678 alone', `grouped ${MASK} alone`],
  ])('masks %s', (input, expected) => {
    expect(maskMoney(input)).toBe(expected);
  });

  it.each([
    'Food is at 85% of the limit',
    'Savings rate 12.5% this month',
    'Due on 13/10/2026 or 2026-10-13',
    'Three trips in 5 days',
    'Option B is better',
    'Spending is 1.8x the usual pace',
    'In 2026 you made 14 transactions',
  ])('leaves %s alone', (input) => {
    expect(maskMoney(input)).toBe(input);
  });
});

describe('maskText', () => {
  it('passes text through when amounts are revealed', () => {
    expect(maskText('Spent 300k', true)).toBe('Spent 300k');
    expect(maskText('Spent 300k', false)).toBe(`Spent ${MASK}`);
  });
});

describe('maskSegments', () => {
  it('Given text with amounts When split Then the amounts are the masked segments and the parts rejoin to the input', () => {
    const text = 'Spent 1.500.000 ₫ on food, 12% more, and 300k on coffee';
    const segments = maskSegments(text);

    expect(segments.filter((s) => s.masked).map((s) => s.text)).toEqual(['1.500.000 ₫', '300k']);
    expect(segments.map((s) => s.text).join('')).toBe(text);
    expect(segments.map((s) => (s.masked ? MASK : s.text)).join('')).toBe(maskMoney(text));
  });

  it('Given text without amounts When split Then it is one plain segment', () => {
    expect(maskSegments('40% of budget')).toEqual([{ text: '40% of budget', masked: false }]);
    expect(maskSegments('')).toEqual([]);
  });
});
