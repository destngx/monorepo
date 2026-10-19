import type { CategoryType } from './accounts';

export interface Category {
  name: string;
  type: CategoryType;
  /** Categories column B; absent when untagged. */
  nature?: CategoryNature;
  /** Categories column D: what belongs here; the AI files transactions by it (WM-CAT-1). */
  definition?: string;
  /** Categories column E: tags that divide the category's spend (WM-CAT-2). */
  splitTags?: string[];
}

/** Body of PATCH /categories/definition; empty values clear them. */
export interface CategoryDefinitionInput {
  category: string;
  definition: string;
  splitTags: string[];
}

/** The month's spend on one of the category's split tags (WM-CAT-3). */
export interface SplitSpend {
  tag: string;
  spent: number;
  percentOfSpent: number;
}

/**
 * Fixed: set by a contract (rent). Essential: needed but can be lowered (groceries). Flexible: wants.
 * Lending: money lent to and repaid by people, neither income nor expense (D30).
 */
export type CategoryNature = 'fixed' | 'essential' | 'flexible' | 'lending';

/** Body of PATCH /categories/nature; 'none' clears the nature. */
export interface CategoryNatureInput {
  category: string;
  nature: CategoryNature | 'none';
}

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
  /** A lending category: shown, but left out of expense and the totals. */
  lending?: boolean;
  /** Day the remaining limit runs out at the current daily burn; null when it will not this month. */
  burnOutDate?: string | null;
  /** Spend on the category's split tags; absent without split tags (WM-CAT-3). */
  splits?: SplitSpend[];
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
