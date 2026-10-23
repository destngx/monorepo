import type { LentLoan, MoneyLentPerson } from '$lib/api/types';
import type { StatusTone } from '$lib/status';

/** People with an open balance (owed either way) and settled people, in the engine's order. */
export function splitPeople(people: MoneyLentPerson[]): { open: MoneyLentPerson[]; settled: MoneyLentPerson[] } {
  return {
    open: people.filter((person) => person.outstanding !== 0),
    settled: people.filter((person) => person.outstanding === 0),
  };
}

/** Who owes whom, as a status badge: owed to you is information, owed by you a warning. */
export function balanceStatus(outstanding: number): { tone: StatusTone; label: string } {
  if (outstanding > 0) return { tone: 'info', label: 'Owes you' };
  if (outstanding < 0) return { tone: 'warning', label: 'You owe' };
  return { tone: 'good', label: 'Settled' };
}

/** Days outstanding in words: days under two months, months under two years, then years. */
export function formatAge(days: number | null | undefined): string {
  if (days === null || days === undefined) return '-';
  if (days < 60) return days === 1 ? '1 day' : `${days} days`;
  if (days < 730) return `${Math.floor(days / 30)} months`;
  return `${(days / 365).toFixed(1)} years`;
}

/** Long-open balances stand out: a warning from 3 months, serious from a year. */
export function ageTone(days: number | null | undefined): StatusTone {
  if (days === null || days === undefined || days < 90) return 'neutral';
  return days < 365 ? 'warning' : 'serious';
}

/** The repayment date from the memo as a badge: overdue is serious, due within 3 days a warning (WM-LEND-6). */
export function repayStatus(daysUntil: number | null | undefined): { tone: StatusTone; label: string } | null {
  if (daysUntil === null || daysUntil === undefined) return null;
  if (daysUntil < 0)
    return { tone: 'serious', label: daysUntil === -1 ? '1 day overdue' : `${-daysUntil} days overdue` };
  if (daysUntil === 0) return { tone: 'warning', label: 'Due today' };
  if (daysUntil === 1) return { tone: 'warning', label: 'Due tomorrow' };
  return { tone: daysUntil <= 3 ? 'warning' : 'neutral', label: `Due in ${daysUntil} days` };
}

/** Open loans not due on the person's soonest repayment date: the ones the Repay by column does not show. */
export function otherOpenLoans(person: MoneyLentPerson): number {
  return person.loans.filter((loan) => loan.outstanding > 0 && loan.repayBy !== person.repayBy).length;
}

/** A loan's label: its Ref # from the sheet, else its place in the loans list (#1 the oldest, WM-LEND-9). */
export function loanLabel(loan: LentLoan, index: number): string {
  return loan.reference ?? `#${index + 1}`;
}

/**
 * The loans each transaction belongs to (`loan` is the place in the loans list, from 1): a lending
 * row is its own loan, a repayment the loans it paid, with the part each got (WM-LEND-8).
 */
export function loanLinks(person: MoneyLentPerson): Map<string, { loan: number; label: string; amount: number }[]> {
  const links = new Map<string, { loan: number; label: string; amount: number }[]>();
  const add = (transactionId: string, loan: number, label: string, amount: number) =>
    links.set(transactionId, [...(links.get(transactionId) ?? []), { loan, label, amount }]);
  person.loans.forEach((loan, index) => {
    const label = loanLabel(loan, index);
    add(loan.transactionId, index + 1, label, loan.amount);
    for (const part of loan.repayments) add(part.transactionId, index + 1, label, part.amount);
  });
  return links;
}
