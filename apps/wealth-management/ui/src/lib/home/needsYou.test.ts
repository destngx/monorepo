import { describe, expect, it } from 'vitest';
import { inboxHeadline, itemAmount } from './needsYou';
import { needsYou, needsYouItem } from './needsYou.testdata';

describe('inboxHeadline', () => {
  it('counts the items and takes the tone of the most severe one', () => {
    expect(inboxHeadline(needsYou())).toEqual({
      tone: 'critical',
      title: '4 things need you',
      text: '1 critical, 1 serious, 1 to watch and 1 suggestion, most urgent first.',
    });
    expect(
      inboxHeadline(needsYou({ items: [needsYouItem({ status: 'info' }), needsYouItem({ status: 'info' })] })),
    ).toEqual({
      tone: 'info',
      title: '2 things need you',
      text: '2 suggestions, most urgent first.',
    });
  });

  it('says nothing needs you, calmly, when the inbox is empty', () => {
    const headline = inboxHeadline(needsYou({ items: [] }));
    expect(headline.tone).toBe('good');
    expect(headline.title).toBe('Nothing needs you');
  });

  it('does not claim all is well when a source could not be checked', () => {
    const unavailable = [
      { part: 'bills' as const, label: 'Bills and card payments' },
      { part: 'goals' as const, label: 'Goals' },
    ];
    expect(inboxHeadline(needsYou({ items: [], unavailable }))).toEqual({
      tone: 'neutral',
      title: 'Nothing needs you in what could be checked',
      text: 'Could not check bills and card payments and goals; try again later.',
    });
    expect(inboxHeadline(needsYou({ items: [needsYouItem()], unavailable: unavailable.slice(0, 1) })).text).toBe(
      '1 serious, most urgent first. Could not check bills and card payments.',
    );
  });
});

describe('itemAmount', () => {
  it('shows what the item is about, a month for goals and a year for estimates', () => {
    expect(itemAmount(needsYouItem())).toEqual({ value: 8_000_000, unit: '' });
    expect(itemAmount(needsYouItem({ kind: 'goal-plan-short', amountVnd: 2_000_000, impactVnd: 24_000_000 }))).toEqual({
      value: 2_000_000,
      unit: 'a month',
    });
    expect(itemAmount(needsYouItem({ kind: 'idle-cash', amountVnd: 50_000_000, impactVnd: 2_250_000 }))).toEqual({
      value: 2_250_000,
      unit: 'a year',
    });
    expect(itemAmount(needsYouItem({ kind: 'recommendation', amountVnd: null, impactVnd: null }))).toBeNull();
    expect(itemAmount(needsYouItem({ kind: 'drafts', amountVnd: null }))).toBeNull();
  });
});
