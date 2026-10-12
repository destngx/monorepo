import type { CaptureBalance, CaptureResult, CaptureRule, CaptureSuggestion, TransactionInput } from '$lib/api/types';

// Review screen state (WM-CAP-8): suggestions become editable rows; rows are
// committed through the batch create path and corrections become rules.

export interface ReviewRow {
  suggestion: CaptureSuggestion;
  selected: boolean;
  tx: TransactionInput;
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
    tx: {
      ...suggestion.transaction,
      date: sheetDateToISO(suggestion.transaction.date),
      tags: [...(suggestion.transaction.tags ?? [])],
      memo: suggestion.transaction.memo ?? '',
      referenceNumber: suggestion.transaction.referenceNumber ?? '',
    },
  }));
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

function round(value: number): number {
  return Math.round(value * 100) / 100;
}

/** Problems that would make the engine reject the row, shown before commit. */
export function rowProblems(tx: TransactionInput, accounts: string[], categories: string[]): string[] {
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

/** Rows to send now: selected and not yet committed, with their indices. */
export function commitBatch(rows: ReviewRow[]): { indices: number[]; inputs: TransactionInput[] } {
  const indices = rows.flatMap((row, index) => (row.selected && !row.createdId ? [index] : []));
  return { indices, inputs: indices.map((index) => cleanInput(rows[index].tx)) };
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
 * chose or the model guessed. Transfers and uncategorised rows teach nothing.
 */
export function learnedRules(rows: ReviewRow[]): CaptureRule[] {
  const rules = new Map<string, CaptureRule>();
  const add = (rule: CaptureRule) => {
    const pattern = rule.pattern.trim();
    if (pattern && rule.value) rules.set(`${rule.kind}:${fold(pattern)}`, { ...rule, pattern });
  };
  for (const { suggestion, tx, createdId } of rows) {
    if (!createdId) continue;
    const hint = suggestion.accountHint?.trim() ?? '';
    if (hint && suggestion.accountSource !== 'alias' && fold(hint) !== fold(tx.accountName)) {
      add({ kind: 'alias', pattern: hint, value: tx.accountName });
    }
    if (suggestion.transferGroup || tx.category === 'Uncategorized') continue;
    const isCorrected = tx.category !== suggestion.transaction.category;
    const isGuessed = suggestion.categorySource === 'ai' || suggestion.categorySource === 'default';
    if (isCorrected || isGuessed) add({ kind: 'payee', pattern: tx.payee, value: tx.category });
  }
  return [...rules.values()];
}

export type ConfidenceLevel = 'high' | 'medium' | 'low';

/** Matches the engine's CaptureLowConfidence (0.7) threshold. */
export function confidenceLevel(confidence: number): ConfidenceLevel {
  if (confidence >= 0.9) return 'high';
  if (confidence >= 0.7) return 'medium';
  return 'low';
}

/** Stable identity of a learned rule, used to track which were saved or skipped. */
export const ruleKey = (rule: CaptureRule) => `${rule.kind}:${rule.pattern.toLowerCase()}=${rule.value}`;
