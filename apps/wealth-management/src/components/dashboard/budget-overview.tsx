'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { BudgetItem } from '@wealth-management/types';
import { Transaction } from '@wealth-management/types';
import { MaskedBalance } from '@/components/ui/masked-balance';
import Link from 'next/link';
import { ArrowRight } from 'lucide-react';
import { getEffectiveDate } from '@wealth-management/utils';

export function BudgetOverview({ budget, transactions }: { budget: BudgetItem[]; transactions: Transaction[] }) {
  const now = new Date();
  const currentMonth = now.getMonth();
  const currentYear = now.getFullYear();

  // Only show expenses in the overview
  const expenseBudget = budget.filter((b) => b.categoryType === 'expense');

  // Construct current month key (e.g. "2026-03")
  const currentMonthKey = `${currentYear}-${String(currentMonth + 1).padStart(2, '0')}`;

  // Map limits and dynamically calculate spent for this current month
  const dynamicBudget = expenseBudget.map((b) => {
    // Determine the exact limit for this month, fallback to average if not specific
    const trueMonthlyLimit = b.monthlyLimits?.[currentMonthKey] || b.monthlyLimit;

    // Only tally transactions that match Category, are in same month/year,
    // and aren't deposits (unless you want to offset somehow, but generally Payment is what touches a budget)
    const categoryTxList = transactions.filter((t) => {
      if (t.category !== b.category) return false;
      const tDate = getEffectiveDate(t);
      return tDate.getMonth() === currentMonth && tDate.getFullYear() === currentYear;
    });

    // Summing pure "payments" out of the category
    // (If a return/refund deposit matches the category, we subtract it from the spent total)
    let spentThisMonth = 0;
    for (const t of categoryTxList) {
      if (t.payment) spentThisMonth += t.payment;
      if (t.deposit) spentThisMonth -= t.deposit; // e.g. a clothing return
    }

    return {
      ...b,
      monthlyLimit: trueMonthlyLimit, // Override display limit with the true month's limit
      monthlySpent: Math.max(0, spentThisMonth), // Don't show negative spending if refunds > buys
    };
  });

  const sortedBudget = [...dynamicBudget]
    .sort((a, b) => {
      const aPercent = a.monthlyLimit > 0 ? a.monthlySpent / a.monthlyLimit : 0;
      const bPercent = b.monthlyLimit > 0 ? b.monthlySpent / b.monthlyLimit : 0;
      return bPercent - aPercent;
    })
    .slice(0, 5);

  return (
    <Card className="col-span-1 shadow-sm flex flex-col justify-between h-full">
      <CardHeader>
        <CardTitle className="text-lg">Budget Overview</CardTitle>
        <CardDescription>Top 5 budget constraints of this month</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {sortedBudget.length === 0 ? (
          <p className="text-sm text-muted-foreground text-center">No budgets set for this month</p>
        ) : (
          sortedBudget.map((cat) => {
            const progress = cat.monthlyLimit > 0 ? Math.min((cat.monthlySpent / cat.monthlyLimit) * 100, 100) : 0;
            const isWarning = progress >= 80 && progress < 100;
            const isDanger = progress >= 100;

            return (
              <div key={cat.category} className="space-y-1">
                <div className="flex justify-between text-sm">
                  <span className="font-medium truncate max-w-[120px]">{cat.category}</span>
                  <span className="text-muted-foreground text-xs tabular-nums">
                    <MaskedBalance amount={cat.monthlySpent} /> / <MaskedBalance amount={cat.monthlyLimit} />
                  </span>
                </div>
                <Progress
                  value={progress}
                  className="h-2"
                  indicatorColor={isDanger ? 'bg-orange-500' : isWarning ? 'bg-amber-500' : 'bg-emerald-500'}
                />
              </div>
            );
          })
        )}
        <div className="pt-4 flex justify-end">
          <Link href="/budget" className="text-sm font-medium flex items-center gap-1 text-primary hover:underline">
            View All <ArrowRight className="h-4 w-4" />
          </Link>
        </div>
      </CardContent>
    </Card>
  );
}
