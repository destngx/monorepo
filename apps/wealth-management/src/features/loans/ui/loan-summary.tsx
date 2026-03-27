"use client";

import { Card, CardContent, CardHeader, CardTitle } from "@wealth-management/ui/card";
import { Landmark, TrendingDown, Clock } from "lucide-react";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { Loan } from "../model/types";

export function LoanSummary({ loans }: { loans: Loan[] }) {
  const totalMonthlyDebt = loans.reduce((sum, l) => sum + l.monthlyDebt, 0);
  const totalMonthlyPaid = loans.reduce((sum, l) => sum + l.monthlyPaid, 0);
  const totalMonthlyRemaining = loans.reduce((sum, l) => sum + l.monthlyRemaining, 0);
  const totalYearlyRemaining = loans.reduce((sum, l) => sum + l.yearlyRemaining, 0);

  const monthProgress = totalMonthlyDebt > 0 ? (totalMonthlyPaid / totalMonthlyDebt) * 100 : 0;

  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card className="bg-blue-50/50 dark:bg-blue-950/20 border-blue-100 dark:border-blue-900 shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-blue-900 dark:text-blue-300">Month Debt</CardTitle>
          <Clock className="h-4 w-4 text-blue-600" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-blue-950 dark:text-blue-100">
            <MaskedBalance amount={totalMonthlyDebt} />
          </div>
          <p className="text-xs text-blue-600/80 dark:text-blue-400 mt-1 flex items-center gap-1">
            Paid: <MaskedBalance amount={totalMonthlyPaid} /> ({Math.round(monthProgress)}%)
          </p>
        </CardContent>
      </Card>

      <Card className="bg-orange-50/50 dark:bg-orange-950/20 border-orange-100 dark:border-orange-900 shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-orange-900 dark:text-orange-300">Remaining (Month)</CardTitle>
          <TrendingDown className="h-4 w-4 text-orange-600" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-orange-950 dark:text-orange-100">
            <MaskedBalance amount={totalMonthlyRemaining} />
          </div>
          <p className="text-xs text-orange-600/80 dark:text-orange-400 mt-1">
            Pending payments for current month
          </p>
        </CardContent>
      </Card>

      <Card className="bg-indigo-50/50 dark:bg-indigo-950/20 border-indigo-100 dark:border-indigo-900 shadow-sm">
        <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
          <CardTitle className="text-sm font-medium text-indigo-900 dark:text-indigo-300">Total Yearly Debt</CardTitle>
          <Landmark className="h-4 w-4 text-indigo-600" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold text-indigo-950 dark:text-indigo-100">
            <MaskedBalance amount={totalYearlyRemaining} />
          </div>
          <p className="text-xs text-indigo-600/80 dark:text-indigo-400 mt-1">
            Remaining balance for the year
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
