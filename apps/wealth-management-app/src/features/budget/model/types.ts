export interface BudgetItem {
  category: string;
  categoryType?: 'income' | 'expense' | 'non-budget';
  monthlyLimit: number;        // average monthly limit (yearlyLimit / 12)
  yearlyLimit: number;
  monthlyLimits?: Record<string, number>; // exact per-month limit keyed "YYYY-MM"
  weeklyLimit?: number;
  monthlySpent: number;
  yearlySpent: number;
  weeklySpent?: number;
  monthlyRemaining: number;
  yearlyRemaining: number;
  weeklyRemaining?: number;
  note?: string;
}
