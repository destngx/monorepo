import type { CategoryType } from './accounts';

export interface Category {
  name: string;
  type: CategoryType;
  /** Categories column B; absent when untagged. */
  nature?: CategoryNature;
}

/** Fixed: set by a contract (rent). Essential: needed but can be lowered (groceries). Flexible: wants. */
export type CategoryNature = 'fixed' | 'essential' | 'flexible';

export interface BudgetItem {
  category: string;
  categoryType?: CategoryType;
  monthlyLimit: number;
  yearlyLimit: number;
  monthlyLimits?: Record<string, number>;
  note?: string;
}

export type BudgetStatus = 'healthy' | 'warning' | 'urgent';

export interface BudgetLine {
  category: string;
  categoryType: CategoryType;
  note?: string;
  limit: number;
  spent: number;
  remaining: number;
  percentUsed: number;
  status: BudgetStatus;
  yearlyLimit: number;
  yearlySpent: number;
  yearlyRemaining: number;
  unbudgeted?: boolean;
  /** Day the remaining limit runs out at the current daily burn; null when it will not this month. */
  burnOutDate?: string | null;
}

/** Six-month budget comparison, "The Wall" (WM-BUD-7). */
export interface BudgetHistory {
  /** Oldest first. */
  months: string[];
  rows: { category: string; values: { month: string; limit: number; spent: number }[] }[];
}

export interface ReallocateInput {
  year: number;
  month: string;
  from: string;
  to: string;
  amount: number;
}

export interface ReallocateResult {
  from: { category: string; limit: number };
  to: { category: string; limit: number };
}

export interface BudgetReport {
  month: string;
  daysInMonth: number;
  daysElapsed: number;
  daysRemaining: number;
  totalLimit: number;
  totalSpent: number;
  totalRemaining: number;
  percentUsed: number;
  dailyBurn: number;
  projectedSpend: number;
  status: BudgetStatus;
  lines: BudgetLine[];
}
