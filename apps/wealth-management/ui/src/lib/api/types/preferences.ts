import type { AllocationTargets } from './investments';

/** Keys of the app-managed Preferences tab (WM-SET-6). */
export type PreferenceKey =
  | 'allocation.target.cash'
  | 'allocation.target.funds'
  | 'allocation.target.crypto'
  | 'goal.emergencyMonths'
  | 'goal.spendingBufferMonths'
  | 'safeToSpend.notReadyAccounts';

/** Keys saved with POST /api/preferences; the account list is set per account. */
export type NumberPreferenceKey = Exclude<PreferenceKey, 'safeToSpend.notReadyAccounts'>;

export interface GoalPoolSettings {
  emergencyMonths: number;
  spendingBufferMonths: number;
}

export interface PreferenceEntry {
  key: PreferenceKey;
  /** The number of a number key; null when unset or a list. */
  value: number | null;
  /** The account list of safeToSpend.notReadyAccounts. */
  names?: string[];
  note: string;
  row: number;
  etag?: string;
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
  entries: PreferenceEntry[];
  /** Rows the engine ignored, e.g. a value out of range. */
  issues: string[];
}

/** A null value clears the key (back to its default). */
export interface PreferenceValue {
  key: NumberPreferenceKey;
  value: number | null;
}
