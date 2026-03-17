"use client";

import { Card, CardContent } from "@/components/ui/card";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { Account } from "@wealth-management/types";
import { Transaction } from "@wealth-management/types";
import { Loan } from "@wealth-management/types";
import { Wallet, Landmark, ArrowLeftRight, Percent } from "lucide-react";
import { getEffectiveDate } from "@wealth-management/utils";

interface SnapshotCardsRowProps {
  accounts: Account[];
  transactions: Transaction[];
  loans: Loan[];
}

export function SnapshotCardsRow({ accounts, transactions, loans }: SnapshotCardsRowProps) {
  const totalLoanLiability = loans.reduce((sum, l) => sum + (l.yearlyRemaining || 0), 0);
  const totalAssets = accounts.filter(a => a.balance > 0).reduce((sum, a) => sum + a.balance, 0);
  const totalLiabilities = accounts.filter(a => a.balance < 0).reduce((sum, a) => sum + Math.abs(a.balance), 0) + totalLoanLiability;

  // Monthly Metrics
  const now = new Date();
  const currentMonthTxns = transactions.filter(t => {
    const txDate = getEffectiveDate(t);
    return txDate.getMonth() === now.getMonth() && txDate.getFullYear() === now.getFullYear();
  });

  const income = currentMonthTxns.reduce((sum, t) => {
    if (t.categoryType === 'income') return sum + (t.deposit || 0);
    return sum;
  }, 0);

  const expense = currentMonthTxns.reduce((sum, t) => {
    if (t.categoryType === 'expense') return sum + (t.payment || 0);
    return sum;
  }, 0);

  const cashFlow = income - expense;
  const savingsRate = income > 0 ? (cashFlow / income) * 100 : 0;

  const stats = [
    { label: "Total Assets", value: totalAssets, icon: Wallet, color: "text-emerald-600", bg: "bg-emerald-50 dark:bg-emerald-950/20" },
    { label: "Total Liabilities", value: totalLiabilities, icon: Landmark, color: "text-rose-600", bg: "bg-rose-50 dark:bg-rose-950/20" },
    { label: "Monthly Cash Flow", value: cashFlow, icon: ArrowLeftRight, color: cashFlow >= 0 ? "text-primary" : "text-orange-600", bg: "bg-zinc-50 dark:bg-zinc-900/50" },
    { label: "Savings Rate", value: savingsRate, icon: Percent, isPercent: true, color: "text-primary", bg: "bg-zinc-50 dark:bg-zinc-900/50" },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      {stats.map((stat, i) => (
        <Card key={i} className="border-none shadow-sm bg-white dark:bg-zinc-950 hover:shadow-md transition-shadow">
          <CardContent className="p-4">
            <div className="flex items-center gap-3">
              <div className={`p-2 rounded-lg ${stat.bg} ${stat.color}`}>
                <stat.icon className="h-4 w-4" />
              </div>
              <div className="space-y-0.5">
                <p className="text-[10px] font-bold uppercase tracking-wider text-muted-foreground">{stat.label}</p>
                <p className={`text-base font-bold ${stat.color}`}>
                  {stat.isPercent ? (
                    `${stat.value.toFixed(1)}%`
                  ) : (
                    <MaskedBalance amount={stat.value} />
                  )}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
