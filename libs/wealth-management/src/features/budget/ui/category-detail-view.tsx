"use client";

import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "-management/ui/card";
import { Transaction } from "@wealth-management/types";
import { formatVND } from "@wealth-management/utils";
import { MaskedBalance } from "-management/ui/masked-balance";
import { Sparkles, ArrowLeft, TrendingUp, History, PieChart as PieChartIcon } from "lucide-react";
import { Button } from "-management/ui/button";
import { CategoryBadge } from "-management/ui/category-badge";
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip } from "recharts";
import { format } from "date-fns";

interface CategoryDetailViewProps {
  category: string;
  limit: number;
  spent: number;
  transactions: Transaction[];
  onBack: () => void;
  onAdjustLimit: (newLimit: number) => void;
}

export function CategoryDetailView({ 
  category, 
  limit, 
  spent, 
  transactions, 
  onBack,
  onAdjustLimit 
}: CategoryDetailViewProps) {
  const remaining = Math.max(0, limit - spent);
  const pieData = [
    { name: "Spent", value: spent, color: "#6366f1" },
    { name: "Remaining", value: remaining, color: "#e2e8f0" }
  ];

  // Spending Timeline (Last 30 days)
  const timelineData = useMemo(() => {
    const days: Record<string, number> = {};
    const today = new Date();
    for (let i = 29; i >= 0; i--) {
      const d = new Date();
      d.setDate(today.getDate() - i);
      days[format(d, "MMM dd")] = 0;
    }
    
    transactions.forEach(t => {
      const d = format(new Date(t.date), "MMM dd");
      if (days[d] !== undefined) {
        days[d] += (t.payment || 0);
      }
    });

    return Object.entries(days).map(([name, amount]) => ({ name, amount }));
  }, [transactions]);

  return (
    <div className="space-y-6">
      <Button variant="ghost" size="sm" onClick={onBack} className="gap-2 group">
        <ArrowLeft className="h-4 w-4 group-hover:-translate-x-1 transition-transform" />
        Back to Overview
      </Button>

      <div className="flex flex-col lg:grid lg:grid-cols-3 gap-6">
        {/* Left Col: Summary & Donut */}
        <div className="space-y-6">
          <Card className="border-none shadow-sm h-full">
            <CardHeader className="pb-2">
              <CategoryBadge category={category} type="expense" className="w-fit" />
            </CardHeader>
            <CardContent className="flex flex-col items-center justify-center pt-6">
              <div className="relative h-[200px] w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="50%"
                      innerRadius={60}
                      outerRadius={80}
                      paddingAngle={5}
                      dataKey="value"
                    >
                      {pieData.map((entry, index) => (
                        <Cell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </Pie>
                  </PieChart>
                </ResponsiveContainer>
                <div className="absolute inset-0 flex flex-col items-center justify-center text-center">
                  <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-widest">Remaining</span>
                  <span className="text-xl font-black text-primary tabular-nums">
                    <MaskedBalance amount={remaining} />
                  </span>
                </div>
              </div>
              <div className="w-full mt-6 space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground font-medium">Monthly Limit</span>
                  <span className="font-bold"><MaskedBalance amount={limit} /></span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground font-medium">Current Spend</span>
                  <span className="font-bold text-indigo-600"><MaskedBalance amount={spent} /></span>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Mid Col: Timeline & Transactions */}
        <div className="lg:col-span-2 space-y-6">
          {/* Spending Timeline */}
          <Card className="border-none shadow-sm">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-bold flex items-center gap-2">
                <TrendingUp className="h-4 w-4 text-primary" />
                Spending Timeline
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="h-[180px] w-full mt-4">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={timelineData}>
                    <XAxis 
                      dataKey="name" 
                      axisLine={false} 
                      tickLine={false} 
                      tick={{ fontSize: 10, fill: '#94a3b8' }} 
                    />
                    <Tooltip 
                      contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 10px 15px -3px rgb(0 0 0 / 0.1)' }}
                      formatter={(v: any) => [formatVND(v || 0), 'Spent']}
                    />
                    <Bar dataKey="amount" fill="#6366f1" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </CardContent>
          </Card>

          {/* AI Insight Panel */}
          <Card className="border-none bg-indigo-50/50 dark:bg-indigo-950/20 border-l-4 border-l-indigo-500">
            <CardContent className="p-6">
               <div className="flex items-start gap-4">
                  <div className="p-2 bg-indigo-500/10 rounded-lg">
                    <Sparkles className="h-5 w-5 text-indigo-500" />
                  </div>
                  <div className="space-y-4">
                    <div>
                      <h3 className="text-sm font-bold text-indigo-900 dark:text-indigo-100 uppercase tracking-tight">AI Category Insight</h3>
                      <p className="text-sm text-indigo-700/80 dark:text-indigo-200/80 leading-relaxed font-medium italic mt-1">
                        "Your average monthly {category} spend over 6 months is {formatVND(4100000)}. Your current budget of {formatVND(limit)} gives you healthy room, but weekend spends are driving 60% of the total."
                      </p>
                    </div>
                    <div className="flex items-center gap-4 p-3 bg-white dark:bg-zinc-900 rounded-xl border border-indigo-100 dark:border-indigo-900/50">
                       <div className="flex-1">
                          <p className="text-[10px] font-black text-muted-foreground uppercase">Suggested Adjustment</p>
                          <p className="text-xs font-bold">Tighten goal to {formatVND(4500000)}</p>
                       </div>
                       <Button size="sm" className="h-8 rounded-lg bg-indigo-600 hover:bg-indigo-700 text-xs font-bold" onClick={() => onAdjustLimit(4500000)}>
                          Apply
                       </Button>
                    </div>
                  </div>
               </div>
            </CardContent>
          </Card>

          {/* Recent Category Transactions */}
          <Card className="border-none shadow-sm overflow-hidden">
            <CardHeader className="pb-2 border-b">
               <CardTitle className="text-sm font-bold flex items-center gap-2">
                 <History className="h-4 w-4 text-primary" />
                 Category Ledger
               </CardTitle>
            </CardHeader>
            <CardContent className="p-0">
               <div className="divide-y max-h-[400px] overflow-y-auto">
                  {transactions.length === 0 ? (
                    <div className="p-12 text-center text-muted-foreground text-sm italic">
                       No transactions yet for this category this month.
                    </div>
                  ) : (
                    transactions.map((t, i) => (
                      <div key={i} className="px-6 py-4 flex justify-between items-center hover:bg-zinc-50 dark:hover:bg-zinc-900 transition-colors">
                        <div className="space-y-0.5">
                          <p className="text-sm font-bold">{t.payee}</p>
                          <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-tight">{format(new Date(t.date), "dd MMM yyyy")}</p>
                        </div>
                        <div className="text-sm font-black tabular-nums">
                           <MaskedBalance amount={t.payment || 0} />
                        </div>
                      </div>
                    ))
                  )}
               </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
