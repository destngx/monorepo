import type { BudgetLine } from '$lib/api/types';

// Move part of one category's monthly limit to another (WM-BUD-7). The engine
// validates too; this mirrors its rules so the dialog can explain them first.

export interface ReallocationDraft {
  from: string;
  to: string;
  amount: number | null;
}

/** Problems with a draft, empty when it can be sent. */
export function reallocationProblems(lines: BudgetLine[], draft: ReallocationDraft): string[] {
  const problems: string[] = [];
  const from = lines.find((line) => line.category === draft.from);
  if (!draft.from || !from) problems.push('Choose the category to take from.');
  if (!draft.to || !lines.some((line) => line.category === draft.to)) problems.push('Choose the category to give to.');
  if (draft.from && draft.from === draft.to) problems.push('Choose two different categories.');
  const amount = draft.amount ?? 0;
  if (!Number.isFinite(amount) || amount <= 0) problems.push('Enter an amount above zero.');
  else if (from && amount > from.limit) problems.push(`The amount is more than the limit of ${from.category}.`);
  return problems;
}

/** Limits after the move, for the dialog's preview. */
export function reallocationPreview(
  lines: BudgetLine[],
  draft: ReallocationDraft,
): { from: number; to: number } | null {
  const from = lines.find((line) => line.category === draft.from);
  const to = lines.find((line) => line.category === draft.to);
  if (!from || !to || reallocationProblems(lines, draft).length) return null;
  const amount = draft.amount ?? 0;
  return { from: from.limit - amount, to: to.limit + amount };
}
