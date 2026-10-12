import { ApiError } from '$lib/api/client';
import type { Transaction, TransactionInput } from '$lib/api/types';

// Faster entry (WM-TX-9): "Save & add another", deep-linked prefills and bulk
// recategorisation.

/** The next blank entry after "Save & add another": same account, date and cleared flag. */
export function nextEntry(saved: TransactionInput): Partial<TransactionInput> {
  return { accountName: saved.accountName, date: saved.date, cleared: saved.cleared };
}

/**
 * A prefill from `/transactions?new=1&account=&payee=&category=`, or null when
 * the URL does not ask for a new transaction. Unknown accounts and categories
 * are dropped so the form never preselects something that cannot be saved.
 */
export function prefillFromURL(
  params: URLSearchParams,
  accounts: string[],
  categories: string[],
): Partial<TransactionInput> | null {
  if (params.get('new') !== '1') return null;
  const prefill: Partial<TransactionInput> = {};
  const account = params.get('account')?.trim();
  const payee = params.get('payee')?.trim();
  const category = params.get('category')?.trim();
  if (account && accounts.includes(account)) prefill.accountName = account;
  if (payee) prefill.payee = payee;
  if (category) {
    const match = categories.find((name) => name.toLowerCase() === category.toLowerCase());
    if (match) prefill.category = match;
  }
  return prefill;
}

/** The writable fields of a loaded row, for a PATCH that changes only some of them. */
export function toInput(tx: Transaction): TransactionInput {
  return {
    accountName: tx.accountName,
    date: tx.date.slice(0, 10),
    referenceNumber: tx.referenceNumber ?? undefined,
    payee: tx.payee,
    tags: [...tx.tags],
    memo: tx.memo ?? undefined,
    category: tx.category,
    cleared: tx.cleared,
    payment: tx.payment ?? undefined,
    deposit: tx.deposit ?? undefined,
  };
}

export interface BulkFailure {
  id: string;
  payee: string;
  message: string;
  conflict: boolean;
}

export interface BulkResult {
  updated: string[];
  failed: BulkFailure[];
}

/**
 * Sets one category on each row, one PATCH at a time with that row's etag, so a
 * row changed in the sheet meanwhile fails alone (409) instead of being overwritten.
 */
export async function recategorise(
  rows: Transaction[],
  category: string,
  update: (id: string, etag: string, input: TransactionInput) => Promise<unknown>,
): Promise<BulkResult> {
  const result: BulkResult = { updated: [], failed: [] };
  for (const tx of rows) {
    if (tx.category === category) {
      result.updated.push(tx.id);
      continue;
    }
    try {
      await update(tx.id, tx.etag, { ...toInput(tx), category });
      result.updated.push(tx.id);
    } catch (error) {
      const isConflict = error instanceof ApiError && error.status === 409;
      const message = isConflict
        ? 'Changed in the sheet since it was loaded; reload and try again.'
        : error instanceof Error
          ? error.message
          : String(error);
      result.failed.push({ id: tx.id, payee: tx.payee, message, conflict: isConflict });
    }
  }
  return result;
}
