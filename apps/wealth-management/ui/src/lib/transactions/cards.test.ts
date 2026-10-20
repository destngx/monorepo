import { describe, expect, it } from 'vitest';
import type { Account, CardConfig, CardHolding } from '$lib/api/types';
import {
  billedCards,
  cardChoices,
  chosenCard,
  mccTag,
  ownTags,
  paidWith,
  settleCardTags,
  withCard,
  type BilledCard,
} from './cards';

const holding = (name: string, keyword: string, extra: Partial<CardHolding> = {}): CardHolding => ({
  name,
  program: 'Shared Programme',
  keyword,
  expiry: '12/2030',
  statementDay: 20,
  dueDay: 5,
  ...extra,
});
const account = (name: string, type: Account['type'] = 'negative active use'): Account => ({
  name,
  dueDate: null,
  creditLimit: null,
  clearedBalance: 0,
  balance: 0,
  type,
  currency: 'VND',
  note: null,
});
const config = (cards: CardHolding[]): CardConfig => ({
  programs: [
    { name: 'Shared Programme', bank: 'Bank A', versions: [] },
    { name: 'Own Programme', bank: 'Bank B', versions: [] },
  ],
  banks: [
    { name: 'Bank A', sharedAccount: 'Card A', referenceLimit: 0 },
    { name: 'Bank B', sharedAccount: 'Card B', referenceLimit: 0 },
  ],
  cards,
  merchants: [],
});

const emerald: BilledCard = { name: 'Emerald', keyword: 'emerald', account: 'Card A', default: true };
const ruby: BilledCard = { name: 'Ruby', keyword: 'ruby', account: 'Card A', default: false, closedFrom: '2026-03-01' };
const silver: BilledCard = { name: 'Silver', keyword: 'silver', account: 'Card Silver', default: false };
const cards = [emerald, ruby, silver];
const row = (date: string, tags: string[] = [], accountName = 'Card A') => ({ accountName, date, tags });

describe('billedCards', () => {
  it('bills a card to the card account named after its keyword, else to its bank shared account', () => {
    const result = billedCards(
      config([
        holding('Emerald', 'Emerald', { default: true }),
        holding('Ruby', 'ruby', { closedFrom: '2026-03-01T00:00:00Z' }),
        holding('Silver', 'silver', { program: 'Own Programme' }),
      ]),
      [account('Card A'), account('Card Silver'), account('Silver savings', 'bank')],
    );

    expect(result).toEqual([
      { name: 'Emerald', keyword: 'emerald', account: 'Card A', default: true },
      { name: 'Ruby', keyword: 'ruby', account: 'Card A', default: false, closedFrom: '2026-03-01' },
      { name: 'Silver', keyword: 'silver', account: 'Card Silver', default: false },
    ]);
  });

  it('skips cards without a keyword', () => {
    expect(billedCards(config([holding('Blank', ' ')]), [])).toEqual([]);
  });
});

describe('ownTags', () => {
  it('keeps the tags the user files by and drops card and MCC tags', () => {
    expect(ownTags(['eating_out', 'Emerald', 'ruby online', 'mcc:5411', 'MCC: 5812'], ['emerald', 'Ruby '])).toEqual([
      'eating_out',
    ]);
  });
});

describe('mccTag', () => {
  it('reads the MCC an mcc:NNNN tag confirms', () => {
    expect(mccTag(['eating_out', ' MCC:5411 '])).toBe('5411');
    expect(mccTag(['mcc:abc'])).toBeUndefined();
    expect(mccTag([])).toBeUndefined();
  });
});

describe('cardChoices', () => {
  it('lists the cards open on the account that day', () => {
    expect(cardChoices(cards, 'Card A', '2026-02-28')).toEqual([emerald, ruby]);
    expect(cardChoices(cards, 'Card A', '2026-03-01T00:00:00Z')).toEqual([emerald]);
    expect(cardChoices(cards, 'Wallet', '2026-02-28')).toEqual([]);
  });
});

describe('chosenCard', () => {
  it('is the tagged card, else the default card the engine counts untagged spend for', () => {
    const open = [emerald, ruby];
    expect(chosenCard(['Ruby'], open)).toBe(ruby);
    expect(chosenCard(['eating_out'], open)).toBe(emerald);
    expect(chosenCard([], [ruby])).toBeUndefined();
  });
});

describe('paidWith', () => {
  it('names the card only while the account had several cards open', () => {
    expect(paidWith(row('2026-02-10', ['ruby']), cards)).toBe(ruby);
    expect(paidWith(row('2026-02-10'), cards)).toBe(emerald);
    expect(paidWith(row('2026-04-10', ['emerald']), cards)).toBeUndefined();
    expect(paidWith(row('2026-04-10', [], 'Card Silver'), cards)).toBeUndefined();
  });

  it('names a card tagged on an account it does not bill to, as the engine counts the row for it', () => {
    expect(paidWith(row('2026-04-10', ['silver'], 'Card A'), cards)).toBe(silver);
  });

  it('prefers the longest keyword a tag contains', () => {
    const gold: BilledCard = { name: 'Gold', keyword: 'gold', account: 'Card A', default: true };
    const rose: BilledCard = { name: 'Rose Gold', keyword: 'rose gold', account: 'Card A', default: false };
    expect(paidWith(row('2026-04-10', ['rose gold']), [gold, rose])).toBe(rose);
  });
});

describe('withCard', () => {
  it('swaps the card tag and keeps the other tags', () => {
    expect(withCard(['eating_out', 'Emerald', 'mcc:5812'], ruby, cards)).toEqual(['eating_out', 'mcc:5812', 'ruby']);
    expect(withCard(['ruby'], ruby, cards)).toEqual(['ruby']);
  });
});

describe('settleCardTags', () => {
  const settle = (tags: string[], account: string, date: string, payment = true, moved = true) =>
    settleCardTags(tags, cards, { account, date, payment, moved });

  it('names the only open card on a payment', () => {
    expect(settle(['eating_out'], 'Card A', '2026-04-10')).toEqual(['eating_out', 'emerald']);
    expect(settle(['eating_out'], 'Card A', '2026-04-10', false)).toEqual(['eating_out']);
  });

  it('leaves the pick to the user while several cards are open', () => {
    expect(settle([], 'Card A', '2026-02-10')).toEqual([]);
    expect(settle(['ruby'], 'Card A', '2026-02-10')).toEqual(['ruby']);
  });

  it('drops a card tag that no longer fits after the account or date moved', () => {
    expect(settle(['ruby', 'eating_out'], 'Wallet', '2026-02-10')).toEqual(['eating_out']);
    expect(settle(['ruby'], 'Card A', '2026-04-10')).toEqual(['emerald']);
  });

  it('keeps the card tags of a row whose account and date did not move', () => {
    expect(settle(['silver'], 'Card A', '2026-02-10', true, false)).toEqual(['silver']);
    expect(settle(['silver'], 'Card A', '2026-04-10', true, false)).toEqual(['silver']);
  });
});
