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

/** Today's expense against a typical day of the 30 days before it (WM-BUD-12). */
export interface SpendingToday {
  date: string;
  spent: number;
  dayToDay: number;
  typicalDay: number;
  changePercent: number | null;
}

/** One day's expense: fixed costs (paid once a month) apart from day-to-day spend. */
export interface SpendingDay {
  date: string;
  fixed: number;
  dayToDay: number;
}

/** One expense category in a spending window. */
export interface SpendingCategory {
  category: string;
  nature?: CategoryNature;
  spent: number;
  percentOfSpent: number;
  monthlyAverage: number;
  previousSpent: number;
  changePercent: number | null;
  splits?: SplitSpend[];
}

/** The expense of a rolling window ending today; averages per day use day-to-day spend only. */
export interface SpendingWindow {
  days: number;
  from: string;
  to: string;
  spent: number;
  fixed: number;
  dayToDay: number;
  dailyAverage: number;
  monthlyAverage: number;
  previous: { from: string; to: string; spent: number; dayToDay: number; dailyAverage: number };
  dailyAverageChangePercent: number | null;
  /** Largest first. */
  categories: SpendingCategory[];
}

export type SpendingRecommendationKind = 'faster' | 'slower' | 'category_rising' | 'split_rising' | 'heavy_day';

/**
 * Engine advice from comparing the windows. Current and usual are daily averages for faster and
 * slower, the last 30 days against the usual 30 days for a rising category or split, and today
 * against a typical day for heavy_day. Amount is the extra (or saving) a month; today's extra for heavy_day.
 */
export interface SpendingRecommendation {
  kind: SpendingRecommendationKind;
  tone: 'warning' | 'info' | 'good';
  category?: string;
  tag?: string;
  nature?: CategoryNature;
  current: number;
  usual: number;
  amount: number;
  changePercent: number;
}

/** How am I spending? Today, the last 30 days and the last 3 months (WM-BUD-12). */
export interface SpendingWindows {
  asOf: string;
  today: SpendingToday;
  /** The last 30 days, then the last 3 months (90 days). */
  windows: SpendingWindow[];
  /** The last 90 days, oldest first. */
  daily: SpendingDay[];
  recommendations: SpendingRecommendation[];
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
