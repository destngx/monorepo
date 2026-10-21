import { withSplitTag } from '$lib/budget/natures';
import type {
  Account,
  CardBankReport,
  CaptureBalance,
  CaptureCategorySource,
  CaptureResult,
  CaptureRule,
  CaptureSuggestion,
  CaptureWrittenRow,
  CategorySuggestion,
  TransactionBatchResult,
  TransactionInput,
} from '$lib/api/types';

// Review screen state (WM-CAP-8): suggestions become editable rows; rows are
// committed through the batch create path and corrections become rules.

export interface ReviewRow {
  suggestion: CaptureSuggestion;
  selected: boolean;
  tx: TransactionInput;
  /** Where the category now comes from: the engine, a re-suggestion after a payee edit, or the user. */
  categorySource?: CaptureCategorySource;
  /** The user's note for later captures from the same counterparty (WM-CAP-24); not written to the sheet. */
  note: string;
  /** rowKey of the row as last saved; drafts only, to tell unsaved edits apart. */
  savedKey?: string;
  /** Set once the row is in the sheet; committed rows are locked. */
  createdId?: string;
  error?: string;
}

/** "31/12/2026" → "2026-12-31"; other values pass through. */
export function sheetDateToISO(value: string): string {
  const match = /^(\d{2})\/(\d{2})\/(\d{4})$/.exec(value.trim());
  return match ? `${match[3]}-${match[2]}-${match[1]}` : value;
}

export function toReviewRows(result: CaptureResult): ReviewRow[] {
  return result.suggestions.map((suggestion) => ({
    suggestion,
    selected: suggestion.selected,
    tx: editableInput(suggestion.transaction),
    categorySource: suggestion.categorySource,
    note: '',
  }));
}

/** A row as the form edits it: ISO date, own tags, empty strings for missing text. */
export function editableInput(tx: TransactionInput): TransactionInput {
  return {
    ...tx,
    date: sheetDateToISO(tx.date),
    tags: [...(tx.tags ?? [])],
    memo: tx.memo ?? '',
    referenceNumber: tx.referenceNumber ?? '',
  };
}

/** Identity of a row's content as saved, so stray spaces are not an edit. */
export const rowKey = (row: ReviewRow) =>
  JSON.stringify({ tx: cleanInput(row.tx), categorySource: row.categorySource ?? '', note: row.note.trim() });

/** True when a saved draft row has edits not saved yet. */
export const isRowDirty = (row: ReviewRow) => row.savedKey !== undefined && rowKey(row) !== row.savedKey;

/**
 * Applies a category suggested for the row's payee, with its split tag
 * (WM-CAT-5). A category the user picked by hand is kept unless forced;
 * transfers keep theirs; only known categories apply.
 */
export function applyCategorySuggestion(
  row: ReviewRow,
  suggestion: CategorySuggestion,
  categories: string[],
  { force = false } = {},
): boolean {
  if (row.suggestion.transferGroup || (row.categorySource === 'manual' && !force)) return false;
  if (!categories.includes(suggestion.category)) return false;
  row.tx.category = suggestion.category;
  row.categorySource = suggestion.source;
  row.tx.tags = withSplitTag(row.tx.tags ?? [], suggestion.splitTag);
  return true;
}

/** Signed effect of a row on its account balance. */
export function rowDelta(tx: TransactionInput): number {
  return (tx.deposit ?? 0) - (tx.payment ?? 0);
}

export interface Reconciliation extends CaptureBalance {
  /** Sum of the selected and committed rows on the account. */
  pending: number;
  projected: number;
  /** balanceAfter - projected: non-zero hints at a missing or duplicated row (WM-CAP-7). */
  difference: number;
}

export function reconcile(balances: CaptureBalance[], rows: ReviewRow[]): Reconciliation[] {
  return balances.map((balance) => {
    const pending = rows
      .filter((row) => (row.selected || row.createdId) && row.tx.accountName === balance.account)
      .reduce((sum, row) => sum + rowDelta(row.tx), 0);
    const projected = balance.sheetBalance + pending;
    return { ...balance, pending, projected, difference: round(balance.balanceAfter - projected) };
  });
}

export interface BalanceProjection {
  account: string;
  currency: string;
  /** Selected rows not yet written on the account; 0 when its drafts are all unticked. */
  rows: number;
  /** The account's sheet balance now. */
  current: number;
  change: number;
  /** The sheet balance once those rows are written. */
  after: number;
}

/**
 * The sheet balance before and after the selected rows, in the account's
 * currency, for each account with a row not yet written: ticking a row
 * moves its account's line instead of adding or removing it. Written rows
 * are already in the reloaded balance; rows on unknown accounts cannot be
 * projected.
 */
export function projectBalances(accounts: Account[], rows: ReviewRow[]): BalanceProjection[] {
  return accounts.flatMap((account) => {
    const open = rows.filter((row) => !row.createdId && row.tx.accountName === account.name);
    if (!open.length) return [];
    const pending = open.filter((row) => row.selected);
    const change = round(pending.reduce((sum, row) => sum + rowDelta(row.tx), 0));
    return [
      {
        account: account.name,
        currency: account.currency,
        rows: pending.length,
        current: account.balance,
        change,
        after: round(account.balance + change),
      },
    ];
  });
}

/**
 * The balance each selected row leaves on its account, in list order (a
 * draft list is oldest first, so it reads like a statement); null for rows
 * not selected, already written or on an unknown account.
 */
export function runningBalances(accounts: Account[], rows: ReviewRow[]): (number | null)[] {
  const balances = new Map(accounts.map((account) => [account.name, account.balance]));
  return rows.map((row) => {
    const balance = balances.get(row.tx.accountName);
    if (!row.selected || row.createdId || balance === undefined) return null;
    const after = round(balance + rowDelta(row.tx));
    balances.set(row.tx.accountName, after);
    return after;
  });
}

function round(value: number): number {
  return Math.round(value * 100) / 100;
}

/** Each account's open card keywords: the tags that say which card paid (WM-CAP-5). */
export function cardTagOptions(reports: CardBankReport[]): Map<string, string[]> {
  const options = new Map<string, string[]>();
  for (const card of reports.flatMap((bank) => bank.cards)) {
    if (card.closedFrom || !card.keyword) continue;
    const list = options.get(card.accountName) ?? [];
    if (!list.includes(card.keyword)) options.set(card.accountName, [...list, card.keyword]);
  }
  return options;
}

const isCardTag = (tag: string, cardTags: string[]) =>
  cardTags.some((option) => option.toLowerCase() === tag.trim().toLowerCase());
const hasCardTag = (tx: TransactionInput, cardTags: string[]) => tx.tags.some((tag) => isCardTag(tag, cardTags));
const allCardTags = (cardTags: Map<string, string[]>) => [...cardTags.values()].flat();

/** The card tag the row names among the account's cards, or '' when none. */
export function pickedCardTag(tx: TransactionInput, cardTags: string[]): string {
  return cardTags.find((option) => tx.tags.some((tag) => isCardTag(tag, [option]))) ?? '';
}

/** Names `cardTag` as the card that paid, replacing any other card tag. */
export function pickCardTag(tx: TransactionInput, cardTag: string, cardTags: Map<string, string[]>): void {
  tx.tags = [...tx.tags.filter((tag) => !isCardTag(tag, allCardTags(cardTags))), cardTag];
}

/** The row's tags without card tags: the user's own, which the tag picker edits. */
export function nonCardTags(tags: string[], cardTags: Map<string, string[]>): string[] {
  const all = allCardTags(cardTags);
  return tags.filter((tag) => !isCardTag(tag, all));
}

/** The card tags a row must pick from: spending on a card account, not transfers or repayments. */
export function requiredCardTags(row: ReviewRow, cardTags: Map<string, string[]>): string[] {
  if (row.suggestion.transferGroup || !((row.tx.payment ?? 0) > 0)) return [];
  return cardTags.get(row.tx.accountName) ?? [];
}

/**
 * After the account changed: drops a card tag of another account's card, as
 * the tag is hidden and would file the row to that card, then adds the card
 * tag when the account has one open card and the row names none.
 */
export function fillCardTag(tx: TransactionInput, cardTags: Map<string, string[]>): void {
  const own = cardTags.get(tx.accountName) ?? [];
  const others = allCardTags(cardTags).filter((tag) => !isCardTag(tag, own));
  tx.tags = tx.tags.filter((tag) => !isCardTag(tag, others));
  if (own.length === 1 && !hasCardTag(tx, own)) tx.tags = [...tx.tags, own[0]];
}

/**
 * Problems that would make the engine reject the row, shown before commit;
 * a row on a card account must also say which card paid.
 */
export function rowProblems(
  tx: TransactionInput,
  accounts: string[],
  categories: string[],
  cardTags: string[] = [],
): string[] {
  const problems: string[] = [];
  if (!tx.accountName) problems.push('Pick an account.');
  else if (!accounts.includes(tx.accountName)) problems.push(`Unknown account "${tx.accountName}".`);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(tx.date)) problems.push('Pick a date.');
  if (!tx.payee.trim()) problems.push('Payee is required.');
  if (!categories.includes(tx.category)) problems.push('Pick a category.');
  const payment = tx.payment ?? 0;
  const deposit = tx.deposit ?? 0;
  if (payment > 0 === deposit > 0) problems.push('Enter either a payment or a deposit.');
  if (payment < 0 || deposit < 0) problems.push('Amounts must be positive.');
  if (cardTags.length && !hasCardTag(tx, cardTags)) problems.push(`Pick the card: ${cardTags.join(' or ')}.`);
  return problems;
}

/** Trims the row and drops empty optionals, as manual entry does. */
export function cleanInput(tx: TransactionInput): TransactionInput {
  const memo = tx.memo?.trim();
  const reference = tx.referenceNumber?.trim();
  return {
    accountName: tx.accountName,
    date: tx.date,
    payee: tx.payee.trim(),
    category: tx.category,
    tags: tx.tags,
    cleared: tx.cleared,
    ...(memo ? { memo } : {}),
    ...(reference ? { referenceNumber: reference } : {}),
    ...(tx.payment ? { payment: tx.payment } : {}),
    ...(tx.deposit ? { deposit: tx.deposit } : {}),
  };
}

/** Rows to send now, with their indices: the selected ones, or only those given; never rows already committed. */
export function commitBatch(rows: ReviewRow[], only?: number[]): { indices: number[]; inputs: TransactionInput[] } {
  const chosen = (row: ReviewRow, index: number) => (only ? only.includes(index) : row.selected);
  const indices = rows.flatMap((row, index) => (chosen(row, index) && !row.createdId ? [index] : []));
  return { indices, inputs: indices.map((index) => cleanInput(rows[index].tx)) };
}

/**
 * The rows a batch wrote, each with the suggestion the engine read, so the
 * engine can learn the payee used for its counterparty (WM-CAP-23). `rows`
 * and `inputs` are the batch, in the order the result's indices refer to.
 */
export function writtenRows(
  rows: ReviewRow[],
  inputs: TransactionInput[],
  outcome: TransactionBatchResult,
): CaptureWrittenRow[] {
  return outcome.rows.flatMap(({ index, id, error }) =>
    id && !error && rows[index] && inputs[index]
      ? [{ suggestion: rows[index].suggestion, transaction: inputs[index], transactionId: id, ...noteOf(rows[index]) }]
      : [],
  );
}

/** The row's note for a request, left out when blank. */
export function noteOf(row: ReviewRow): { note?: string } {
  const note = row.note.trim();
  return note ? { note } : {};
}

const fold = (value: string) =>
  value
    .normalize('NFD')
    .replace(/\p{M}/gu, '')
    .replace(/đ/gi, 'd')
    .toLowerCase()
    .replace(/[^\p{L}\p{N}]+/gu, ' ')
    .trim();

/**
 * Rules learned from committed rows (WM-CAP-11): an account hint the engine
 * did not resolve through an alias, and a payee whose category the user
 * chose or the model guessed, also after a payee edit. Transfers and uncategorised rows teach nothing.
 */
export function learnedRules(rows: ReviewRow[]): CaptureRule[] {
  const rules = new Map<string, CaptureRule>();
  const add = (rule: CaptureRule) => {
    const pattern = rule.pattern.trim();
    if (pattern && rule.value) rules.set(`${rule.kind}:${fold(pattern)}`, { ...rule, pattern });
  };
  for (const { suggestion, tx, createdId, categorySource } of rows) {
    if (!createdId) continue;
    const hint = suggestion.accountHint?.trim() ?? '';
    if (hint && suggestion.accountSource !== 'alias' && fold(hint) !== fold(tx.accountName)) {
      add({ kind: 'alias', pattern: hint, value: tx.accountName });
    }
    if (suggestion.transferGroup || tx.category === 'Uncategorized') continue;
    const isCorrected = tx.category !== suggestion.transaction.category;
    const isGuessed = categorySource === 'ai' || categorySource === 'default';
    if (isCorrected || isGuessed) add({ kind: 'payee', pattern: tx.payee, value: tx.category });
  }
  return [...rules.values()];
}

/** Stable identity of a learned rule, used to track which were saved or skipped. */
export const ruleKey = (rule: CaptureRule) => `${rule.kind}:${rule.pattern.toLowerCase()}=${rule.value}`;
