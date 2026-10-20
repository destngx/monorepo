import type { Account, CardConfig } from '$lib/api/types';

/**
 * Which card paid is kept as the card's keyword in the row's tags (WM-RULE-7),
 * for the cashback engine. The ledger shows it apart from the user's own tags,
 * and only where the account bills more than one card (WM-TX-3).
 */
export interface BilledCard {
  name: string;
  /** Lower-cased: a tag containing it names the card. */
  keyword: string;
  /** The account the card bills to. */
  account: string;
  default: boolean;
  /** yyyy-mm-dd: the first day the card takes no payments. */
  closedFrom?: string;
}

const day = (iso: string) => iso.slice(0, 10);
const MCC_TAG = /^mcc:\s*(\d+)$/;

/**
 * Held cards with their billing account, as the engine finds it: a card
 * account named after the keyword, else the bank's shared account.
 */
export function billedCards(config: CardConfig, accounts: Account[]): BilledCard[] {
  return config.cards.flatMap((card) => {
    const keyword = card.keyword.trim().toLowerCase();
    if (!keyword) return [];
    const bank = config.programs.find((program) => program.name === card.program)?.bank;
    const shared = config.banks.find((candidate) => candidate.name === bank)?.sharedAccount ?? '';
    const own = accounts.find(
      (account) => account.type === 'negative active use' && account.name.toLowerCase().includes(keyword),
    );
    const billed: BilledCard = { name: card.name, keyword, account: own?.name ?? shared, default: !!card.default };
    if (card.closedFrom) billed.closedFrom = day(card.closedFrom);
    return [billed];
  });
}

/** The card a tag names: the longest keyword it contains. */
function cardOfTag(tag: string, cards: BilledCard[]): BilledCard | undefined {
  const lower = tag.toLowerCase();
  let match: BilledCard | undefined;
  for (const card of cards) {
    if (lower.includes(card.keyword) && (!match || card.keyword.length > match.keyword.length)) match = card;
  }
  return match;
}

function taggedCard(tags: string[], cards: BilledCard[]): BilledCard | undefined {
  for (const tag of tags) {
    const card = cardOfTag(tag, cards);
    if (card) return card;
  }
  return undefined;
}

/** The MCC an `mcc:NNNN` tag confirms for the payment (C4). */
export function mccTag(tags: string[]): string | undefined {
  for (const tag of tags) {
    const code = MCC_TAG.exec(tag.trim().toLowerCase())?.[1];
    if (code) return code;
  }
  return undefined;
}

/** The tags the user files by, without the card tags (any containing a card keyword) and MCC tags the engine reads. */
export function ownTags(tags: string[], keywords: string[]): string[] {
  const words = keywords.map((keyword) => keyword.trim().toLowerCase()).filter(Boolean);
  return tags.filter((tag) => {
    const lower = tag.trim().toLowerCase();
    return !MCC_TAG.test(lower) && !words.some((word) => lower.includes(word));
  });
}

/** The account's cards open on the date: the ones a payment that day can name. */
export function cardChoices(cards: BilledCard[], account: string, date: string): BilledCard[] {
  return cards.filter((card) => card.account === account && !(card.closedFrom && day(date) >= card.closedFrom));
}

/** The card a row counts for among its choices: the tagged one, else the default card. */
export function chosenCard(tags: string[], choices: BilledCard[]): BilledCard | undefined {
  return taggedCard(tags, choices) ?? choices.find((card) => card.default);
}

/**
 * The card to name on a ledger row: only when the account had several cards
 * open that day, or the tag names a card the account does not bill to.
 */
export function paidWith(
  tx: { accountName: string; date: string; tags: string[] },
  cards: BilledCard[],
): BilledCard | undefined {
  const tagged = taggedCard(tx.tags, cards);
  if (tagged && tagged.account !== tx.accountName) return tagged;
  const choices = cardChoices(cards, tx.accountName, tx.date);
  return choices.length > 1 ? chosenCard(tx.tags, choices) : undefined;
}

/** The tags with `card` as the card that paid, replacing any other card's tag. */
export function withCard(tags: string[], card: BilledCard, cards: BilledCard[]): string[] {
  if (taggedCard(tags, cards) === card) return tags;
  return [...tags.filter((tag) => !cardOfTag(tag, cards)), card.keyword];
}

/**
 * The tags to save. A card tag left from before the account or date changed
 * (`moved`) is dropped when the card is not open there; a payment on an
 * account with one open card and no card tag names it, as Smart Capture does (WM-CAP-5), so
 * it counts for that card even when no card is marked default.
 */
export function settleCardTags(
  tags: string[],
  cards: BilledCard[],
  row: { account: string; date: string; payment: boolean; moved: boolean },
): string[] {
  const choices = cardChoices(cards, row.account, row.date);
  const tagged = taggedCard(tags, cards);
  let settled = tags;
  if (row.moved && tagged && !choices.includes(tagged)) settled = tags.filter((tag) => !cardOfTag(tag, cards));
  if (row.payment && choices.length === 1 && !taggedCard(settled, cards)) settled = [...settled, choices[0].keyword];
  return settled;
}
