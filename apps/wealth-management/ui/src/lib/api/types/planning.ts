import type { Transaction } from './transactions';

export interface Goal {
  id: string;
  name: string;
  type: string;
  emoji: string;
  targetAmount: number;
  currentAmount: number;
  deadline: string;
  status: 'ON_TRACK' | 'AT_RISK' | 'OFF_TRACK';
}

/** One person's money-lent ledger (WM-LEND-1). */
export interface MoneyLentPerson {
  /** The most recent spelling of the payee. */
  name: string;
  lent: number;
  repaid: number;
  /** lent - repaid: positive is owed to you, negative is owed by you, 0 is settled. */
  outstanding: number;
  /** yyyy-mm-dd of the latest transaction. */
  lastActivity: string;
  /** yyyy-mm-dd the open balance started; null when settled. */
  openSince: string | null;
  daysOutstanding: number | null;
  /** Newest first. */
  transactions: Transaction[];
}

/** Money lent per person from lending-category transactions (WM-LEND-1..3, D30). */
export interface MoneyLent {
  asOf: string;
  owedToYou: number;
  youOwe: number;
  peopleOwing: number;
  peopleYouOwe: number;
  /** Longest open balance owed to you; null when nobody owes you. */
  oldestDaysOutstanding: number | null;
  /** Owed to you first, then owed by you, then settled. */
  people: MoneyLentPerson[];
}
