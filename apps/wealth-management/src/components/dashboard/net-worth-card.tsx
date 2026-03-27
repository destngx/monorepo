"use client";

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@wealth-management/ui/card";
import { MaskedBalance } from "@/components/ui/masked-balance";
import { Account } from "@wealth-management/types";
import { Transaction } from "@wealth-management/types";
import { Loan } from "@wealth-management/types";
import { TrendingUp } from "lucide-react";
import { format, subMonths } from "date-fns";
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from "recharts";

interface NetWorthCardProps {
  accounts: Account[];
  transactions: Transaction[];
  loans: Loan[];
}

export function NetWorthCard({ accounts, transactions, loans }: NetWorthCardProps) {
  const totalLoanLiability = loans.reduce((sum, l) => sum + l.yearlyRemaining, 0);
  const totalNetWorth = accounts.reduce((sum, a) => sum + a.balance, 0) - totalLoanLiability;

  // Group by positive/negative for display
  const assets = accounts.filter(a => a.balance > 0).reduce((sum, a) => sum + a.balance, 0);
  const liabilities = accounts.filter(a => a.balance < 0).reduce((sum, a) => sum + Math.abs(a.balance), 0) + totalLoanLiability;

  // Chart Logic
  const now = new Date();
  
  // Calculate net flow for the current month up to today
  const currentMonthTxns = transactions.filter(t => {
    const txDate = new Date(t.date);
    return txDate.getMonth() === now.getMonth() && txDate.getFullYear() === now.getFullYear();
  });
  currentMonthTxns.reduce((sum, t) => sum + (t.deposit || 0) - (t.payment || 0), 0);

  // We track the running balance backwards
  let runningBalance = totalNetWorth;
  
  // Create an array of data points for the last 6 months
  const chartData = [];
  
  for (let i = 0; i < 6; i++) {
    const d = subMonths(now, i);
    
    // We want the balance at the END of this month 'd'
    if (i > 0) {
      const nextMonth = subMonths(now, i - 1);
      const nextMonthTxns = transactions.filter(t => {
        const txDate = new Date(t.date);
        return txDate.getMonth() === nextMonth.getMonth() && txDate.getFullYear() === nextMonth.getFullYear();
      });
      const nextMonthFlow = nextMonthTxns.reduce((sum, t) => sum + (t.deposit || 0) - (t.payment || 0), 0);
      runningBalance = runningBalance - nextMonthFlow;
    }

    const monthKey = format(d, "MMM yyyy");
    
    chartData.unshift({
      name: monthKey,
      Balance: runningBalance
    });
  }

  return (
    <Card className="col-span-full md:col-span-2 shadow bg-gradient-to-br from-card to-card/50 h-full flex flex-col">
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <div>
          <CardTitle className="text-xl font-medium text-muted-foreground">Total Net Worth</CardTitle>
          <CardDescription className="text-xs mt-1 border-b-0">Current balance & historical trend</CardDescription>
        </div>
        <TrendingUp className="h-6 w-6 text-emerald-500" />
      </CardHeader>
      <CardContent className="flex-1 flex flex-col">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 flex-1">
          {/* Values Column */}
          <div className="md:col-span-1 flex flex-col justify-center">
            <div className="text-2xl font-bold mb-6 italic tracking-tight">
              <MaskedBalance amount={totalNetWorth} />
            </div>
            <div className="flex flex-col gap-4 text-sm">
              <div className="flex flex-col gap-1 p-3 bg-muted rounded-lg shadow-sm">
                <span className="text-muted-foreground font-medium">Total Assets</span>
                <MaskedBalance amount={assets} className="font-semibold text-lg text-emerald-600" />
              </div>
              <div className="flex flex-col gap-1 p-3 bg-muted rounded-lg shadow-sm">
                <span className="text-muted-foreground font-medium">Liabilities</span>
                <MaskedBalance amount={liabilities} className="font-semibold text-lg text-orange-500" />
              </div>
            </div>
          </div>
          
          {/* Chart Column */}
          <div className="md:col-span-2 h-64 md:h-full min-h-[250px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="colorBalanceCurrent" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3}/>
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e5e7eb" />
                <XAxis dataKey="name" fontSize={12} tickLine={false} axisLine={false} />
                <YAxis 
                  fontSize={12} 
                  tickLine={false} 
                  axisLine={false} 
                  tickFormatter={(value) => `${(value / 1000000).toFixed(0)}M`}
                  domain={['auto', 'auto']}
                />
                <Tooltip 
                  formatter={(value: any) => new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(value)}
                  contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
                />
                <Area type="monotone" dataKey="Balance" stroke="#3b82f6" strokeWidth={3} fillOpacity={1} fill="url(#colorBalanceCurrent)" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
