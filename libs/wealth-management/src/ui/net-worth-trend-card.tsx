"use client";

import { Card, CardContent } from "./card";
import { MaskedBalance } from "./masked-balance";
import { Account } from "@wealth-management/types";
import { Transaction } from "@wealth-management/types";
import { Loan } from "@wealth-management/types";
import { TrendingUp, TrendingDown } from "lucide-react";
import { subMonths, subDays } from "date-fns";
import { AreaChart, Area, ResponsiveContainer, YAxis } from "recharts";
import { useMemo } from "react";

interface NetWorthTrendCardProps {
  accounts: Account[];
  transactions: Transaction[];
  loans: Loan[];
}

export function NetWorthTrendCard({ accounts, transactions, loans }: NetWorthTrendCardProps) {
  const totalLoanLiability = loans.reduce((sum, l) => sum + (l.yearlyRemaining || 0), 0);
  const currentNetWorth = accounts.reduce((sum, a) => sum + (a.balance || 0), 0) - totalLoanLiability;

  // Calculate MoM Change
  const now = new Date();
  subMonths(now, 1);
  
  const currentMonthTxns = transactions.filter(t => {
    const d = new Date(t.date);
    return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
  });
  const currentMonthFlow = currentMonthTxns.reduce((sum, t) => sum + (t.deposit || 0) - (t.payment || 0), 0);
  const lastMonthNetWorth = currentNetWorth - currentMonthFlow;
  
  const momChange = lastMonthNetWorth !== 0 ? ((currentNetWorth - lastMonthNetWorth) / Math.abs(lastMonthNetWorth)) * 100 : 0;
  const isPositive = momChange >= 0;

  // Generate Sparkline Data (last 30 days)
  const sparklineData = useMemo(() => {
    const data = [];
    let runningNetWorth = currentNetWorth;
    
    // We go backwards from today
    for (let i = 0; i < 30; i++) {
      const targetDate = subDays(now, i);
      const dayTxns = transactions.filter(t => {
        const d = new Date(t.date);
        return d.toDateString() === targetDate.toDateString();
      });
      const dayFlow = dayTxns.reduce((sum, t) => sum + (t.deposit || 0) - (t.payment || 0), 0);
      
      data.unshift({ value: runningNetWorth });
      runningNetWorth -= dayFlow;
    }
    return data;
  }, [currentNetWorth, transactions]);

  return (
    <Card className="overflow-hidden border-none bg-gradient-to-br from-zinc-900 to-black text-white shadow-2xl relative">
      <div className="absolute top-0 right-0 w-64 h-64 bg-primary/10 rounded-full blur-3xl -mr-32 -mt-32 pointer-events-none" />
      
      <CardContent className="p-8 relative z-10">
        <div className="flex flex-col md:flex-row md:items-end justify-between gap-6">
          <div className="space-y-2">
            <p className="text-zinc-400 text-sm font-medium uppercase tracking-widest">Total Net Worth</p>
            <div className="flex items-baseline gap-4">
              <h2 className="text-5xl font-black tracking-tighter">
                <MaskedBalance amount={currentNetWorth} />
              </h2>
              <div className={`flex items-center gap-1 text-sm font-bold px-2 py-1 rounded-full ${
                isPositive ? 'bg-emerald-500/20 text-emerald-400' : 'bg-rose-500/20 text-rose-400'
              }`}>
                {isPositive ? <TrendingUp className="h-3 w-3" /> : <TrendingDown className="h-3 w-3" />}
                {isPositive ? '+' : ''}{momChange.toFixed(1)}%
              </div>
            </div>
            <p className="text-zinc-500 text-xs">Updated just now · Net growth this month: <span className={isPositive ? 'text-emerald-400' : 'text-rose-400'}><MaskedBalance amount={Math.abs(currentMonthFlow)} /></span></p>
          </div>

          {/* Sparkline */}
          <div className="w-full md:w-48 h-20 opacity-80 hover:opacity-100 transition-opacity min-w-0 min-h-0">
            <ResponsiveContainer width="100%" height="100%" minWidth={0}>
              <AreaChart data={sparklineData}>
                <defs>
                  <linearGradient id="sparklineColor" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={isPositive ? "#10b981" : "#f43f5e"} stopOpacity={0.3}/>
                    <stop offset="95%" stopColor={isPositive ? "#10b981" : "#f43f5e"} stopOpacity={0}/>
                  </linearGradient>
                </defs>
                <YAxis hide domain={['dataMin - 1000000', 'dataMax + 1000000']} />
                <Area 
                  type="monotone" 
                  dataKey="value" 
                  stroke={isPositive ? "#10b981" : "#f43f5e"} 
                  strokeWidth={2} 
                  fill="url(#sparklineColor)" 
                  animationDuration={2000}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
