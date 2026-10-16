import type { MoneyLentPerson } from '$lib/api/types';
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
