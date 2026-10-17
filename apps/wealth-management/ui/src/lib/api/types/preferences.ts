import type { AllocationTargets } from './investments';

/** Keys of the app-managed Preferences tab (WM-SET-6). */
export type PreferenceKey =
  | 'allocation.target.cash'
  | 'allocation.target.funds'
  | 'allocation.target.crypto'
  | 'goal.emergencyMonths'
  | 'goal.spendingBufferMonths'
  | 'safeToSpend.notReadyAccounts'
  | 'idleCash.dayToDayFloat'
  | 'idleCash.emergencyAccounts'
  | 'accounts.minimumBalance';

/** Keys saved with POST /api/preferences; the account lists are set per account. */
export type NumberPreferenceKey = Exclude<
  PreferenceKey,
  'safeToSpend.notReadyAccounts' | 'idleCash.emergencyAccounts' | 'accounts.minimumBalance'
>;

export interface GoalPoolSettings {
  emergencyMonths: number;
  spendingBufferMonths: number;
}

export interface PreferenceEntry {
  key: PreferenceKey;
  /** The number of a number key; null when unset or a list. */
  value: number | null;
  /** The account list of an account list key. */
  names?: string[];
  /** The per-account minimums of accounts.minimumBalance. */
  minimums?: AccountMinimum[];
  note: string;
  row: number;
  etag?: string;
}

/** Idle cash settings (D34): the day-to-day float and the emergency accounts. */
export interface IdleCashSettings {
  /** VND; null = the spending buffer. */
  dayToDayFloat: number | null;
  emergencyAccounts: string[];
  etag: string;
}

/** The balance to keep in one account (D35), e.g. for a bank reward on the average monthly balance. */
export interface AccountMinimum {
  account: string;
  minimum: number;
  note?: string;
}

/** The saved minimum balances, with the ETag a change must send. */
export interface AccountMinimumSettings {
  accounts: AccountMinimum[];
  etag: string;
}

/** Accounts left out of safe to spend's ready money (D33). */
export interface SafeToSpendSettings {
  notReadyAccounts: string[];
  etag: string;
}

export interface Preferences {
  /** False until the first save creates the tab. */
  tabExists: boolean;
  /** Null until all three targets are set and add up to 100. */
  allocationTargets: AllocationTargets | null;
  goalPool: GoalPoolSettings;
  safeToSpend: SafeToSpendSettings;
  idleCash: IdleCashSettings;
  accountMinimums: AccountMinimumSettings;
  entries: PreferenceEntry[];
  /** Rows the engine ignored, e.g. a value out of range. */
  issues: string[];
}

/** A null value clears the key (back to its default). */
export interface PreferenceValue {
  key: NumberPreferenceKey;
  value: number | null;
}
