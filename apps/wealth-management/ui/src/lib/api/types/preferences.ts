import type { AllocationTargets } from './investments';

/** Keys of the app-managed Preferences tab (WM-SET-6). */
export type PreferenceKey =
  | 'allocation.target.cash'
  | 'allocation.target.funds'
  | 'allocation.target.crypto'
  | 'goal.emergencyMonths'
  | 'goal.spendingBufferMonths';

export interface GoalPoolSettings {
  emergencyMonths: number;
  spendingBufferMonths: number;
}

export interface PreferenceEntry {
  key: PreferenceKey;
  value: number | null;
  note: string;
  row: number;
}

export interface Preferences {
  /** False until the first save creates the tab. */
  tabExists: boolean;
  /** Null until all three targets are set and add up to 100. */
  allocationTargets: AllocationTargets | null;
  goalPool: GoalPoolSettings;
  entries: PreferenceEntry[];
  /** Rows the engine ignored, e.g. a value out of range. */
  issues: string[];
}

/** A null value clears the key (back to its default). */
export interface PreferenceValue {
  key: PreferenceKey;
  value: number | null;
}
