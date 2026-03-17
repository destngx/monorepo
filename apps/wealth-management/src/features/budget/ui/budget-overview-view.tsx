'use client';

import { Card, CardContent, CardHeader } from '@/components/ui/card';
import { Progress } from '@/components/ui/progress';
import { BudgetItem } from '../model/types';
import { MaskedBalance } from '@/components/ui/masked-balance';
import { Sparkles, ChevronLeft, ChevronRight, Plus, TrendingUp } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { CategoryBadge } from '@/components/ui/category-badge';

interface BudgetOverviewViewProps {
  budget: BudgetItem[];
  date: Date;
  onPrev: () => void;
  onNext: () => void;
  onReset: () => void;
  onSelectCategory: (category: string) => void;
  briefing?: string;
  detailedBrief?: string;
  insights?: { forecastedSavings: number; unusualVelocity: string };
  isLoadingAdvisor?: boolean;
}

export function BudgetOverviewView({
  budget,
  date,
  onPrev,
  onNext,
  onReset,
  onSelectCategory,
  briefing: _briefing,
  detailedBrief,
  insights,
  isLoadingAdvisor,
}: BudgetOverviewViewProps) {
  const expenseBudget = budget.filter((b) => b.categoryType === 'expense');
  const totalLimit = expenseBudget.reduce((sum, item) => sum + item.monthlyLimit, 0);
  const totalSpent = expenseBudget.reduce((sum, item) => sum + item.monthlySpent, 0);
  const totalProgress = totalLimit > 0 ? (totalSpent / totalLimit) * 100 : 0;

  // Calculate local burn rate
  const now = new Date();
  const isCurrentMonth = date.getMonth() === now.getMonth() && date.getFullYear() === now.getFullYear();
  const daysPassed = isCurrentMonth ? now.getDate() : 30; // Approximation for historical
  const dailyBurnRate = Math.round(totalSpent / daysPassed);

  const getProgressColor = (percent: number) => {
    if (percent >= 90) return 'bg-rose-500';
    if (percent >= 70) return 'bg-amber-500';
    return 'bg-emerald-500';
  };

  return (
    <div className="space-y-6">
      {/* Month Navigation */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2 bg-muted/50 p-1 rounded-lg border">
          <Button variant="ghost" size="icon" onClick={onPrev} className="h-8 w-8">
            <ChevronLeft className="h-4 w-4" />
          </Button>
          <div className="px-4 text-sm font-bold min-w-[120px] text-center">
            {date.toLocaleDateString('vi-VN', { month: 'long', year: 'numeric' })}
          </div>
          <Button variant="ghost" size="icon" onClick={onNext} className="h-8 w-8">
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
        <Button variant="outline" size="sm" onClick={onReset} className="text-xs font-bold">
          Today
        </Button>
      </div>

      {/* AI Insight Summary Strip */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {isLoadingAdvisor || !insights ? (
          <>
            {[1, 2].map((i) => (
              <Card key={i} className="border-none bg-zinc-50 dark:bg-zinc-900 shadow-sm animate-pulse">
                <CardContent className="p-4 flex items-center gap-3">
                  <div className="h-8 w-8 bg-zinc-200 dark:bg-zinc-800 rounded-lg" />
                  <div className="space-y-2">
                    <div className="h-2 w-16 bg-zinc-200 dark:bg-zinc-800 rounded" />
                    <div className="h-4 w-24 bg-zinc-200 dark:bg-zinc-800 rounded" />
                  </div>
                </CardContent>
              </Card>
            ))}
          </>
        ) : (
          <>
            <Card className="border-none bg-zinc-50 dark:bg-zinc-900 shadow-sm">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-2 bg-emerald-500/10 rounded-lg">
                  <Plus className="h-4 w-4 text-emerald-500" />
                </div>
                <div>
                  <p className="text-[10px] font-black uppercase text-muted-foreground tracking-tighter">
                    Forecasted Savings
                  </p>
                  <p className="text-sm font-bold text-zinc-800 dark:text-zinc-200">
                    <MaskedBalance amount={insights.forecastedSavings} />
                  </p>
                </div>
              </CardContent>
            </Card>
            <Card className="border-none bg-zinc-50 dark:bg-zinc-900 shadow-sm">
              <CardContent className="p-4 flex items-center gap-3">
                <div className="p-2 bg-amber-500/10 rounded-lg">
                  <Sparkles className="h-4 w-4 text-amber-500" />
                </div>
                <div>
                  <p className="text-[10px] font-black uppercase text-muted-foreground tracking-tighter">
                    Unusual Velocity
                  </p>
                  <p className="text-sm font-bold text-zinc-800 dark:text-zinc-200">{insights.unusualVelocity}</p>
                </div>
              </CardContent>
            </Card>
          </>
        )}
      </div>

      {/* AI Detailed Briefing */}
      {isLoadingAdvisor ? (
        <div className="bg-blue-50/50 dark:bg-blue-950/20 border border-blue-100 dark:border-blue-900/50 rounded-xl p-5 flex gap-4 animate-pulse">
          <div className="h-8 w-8 bg-blue-100 dark:bg-blue-900 rounded-lg" />
          <div className="space-y-2 flex-1">
            <div className="h-2 w-24 bg-blue-100 dark:bg-blue-900 rounded" />
            <div className="h-3 w-3/4 bg-blue-100 dark:bg-blue-900 rounded" />
            <div className="h-3 w-1/2 bg-blue-100 dark:bg-blue-900 rounded" />
          </div>
        </div>
      ) : (
        detailedBrief && (
          <div className="bg-blue-50/50 dark:bg-blue-950/20 border border-blue-100 dark:border-blue-900/50 rounded-xl p-5 flex gap-4">
            <div className="p-2 bg-blue-500/10 rounded-lg h-fit">
              <Sparkles className="h-4 w-4 text-blue-600" />
            </div>
            <div className="space-y-1">
              <p className="text-[10px] font-black uppercase text-blue-600/60 tracking-wider">
                Advisor's Executive Summary
              </p>
              <p className="text-sm font-medium text-blue-900 dark:text-blue-100 leading-relaxed italic">
                "{detailedBrief}"
              </p>
            </div>
          </div>
        )
      )}

      {/* Total Budget Progress Bar */}
      <Card className="border-none shadow-sm overflow-hidden relative">
        <CardContent className="p-6 space-y-4">
          <div className="flex justify-between items-end">
            <div className="space-y-1">
              <span className="text-[10px] font-black uppercase tracking-tighter text-muted-foreground">
                Master Budget Progress
              </span>
              <div className="text-3xl font-black tabular-nums flex items-baseline gap-2">
                {Math.round(totalProgress)}%
                {totalProgress > 100 && (
                  <span className="text-[10px] text-rose-500 font-black animate-pulse uppercase">Budget Burst</span>
                )}
              </div>
            </div>
            <div className="text-right">
              <div className="text-sm font-bold">
                <MaskedBalance amount={totalSpent} />
              </div>
              <div className="text-[10px] text-muted-foreground font-medium">
                of <MaskedBalance amount={totalLimit} /> budget
              </div>
            </div>
          </div>

          <div className="space-y-2">
            <Progress
              value={Math.min(totalProgress, 100)}
              indicatorColor={getProgressColor(totalProgress)}
              className="h-3 bg-zinc-100 dark:bg-zinc-800"
            />

            {/* Velocity Metric Integration */}
            <div className="flex items-center justify-between text-[10px] font-bold">
              <div className="flex items-center gap-1.5 text-zinc-500">
                <div className="p-1 bg-zinc-100 dark:bg-zinc-800 rounded">
                  <TrendingUp className="h-3 w-3 text-indigo-500" />
                </div>
                <span>
                  Daily Burn:{' '}
                  <span className="text-zinc-900 dark:text-zinc-100">
                    <MaskedBalance amount={dailyBurnRate} />
                  </span>{' '}
                  / day
                </span>
              </div>
              <div className="text-zinc-400 italic">
                {isCurrentMonth ? `${30 - daysPassed} days remaining` : 'Period Closed'}
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Category List */}
      <Card className="border-none shadow-sm overflow-hidden">
        <CardHeader className="pb-2 border-b">
          <div className="flex justify-between items-center text-[10px] font-black uppercase tracking-widest text-muted-foreground">
            <span>Category</span>
            <div className="flex gap-8">
              <span className="w-24 text-right">Spent</span>
              <span className="w-24 text-right">Limit</span>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <div className="divide-y">
            {expenseBudget.map((item, i) => {
              const progress = item.monthlyLimit > 0 ? (item.monthlySpent / item.monthlyLimit) * 100 : 0;
              return (
                <div
                  key={i}
                  className="px-6 py-4 flex flex-col gap-3 hover:bg-zinc-50 dark:hover:bg-zinc-900 transition-colors cursor-pointer group"
                  onClick={() => onSelectCategory(item.category)}
                >
                  <div className="flex justify-between items-center">
                    <div className="flex items-center gap-3">
                      <CategoryBadge category={item.category} type="expense" />
                      <span className="text-[10px] font-bold text-zinc-400 group-hover:text-primary transition-colors">
                        {progress > 90 ? '↑ Urgent' : progress > 70 ? '↗ Warning' : '→ Healthy'}
                      </span>
                    </div>
                    <div className="flex gap-8 tabular-nums font-bold text-sm">
                      <div className="w-24 text-right">
                        <MaskedBalance amount={item.monthlySpent} />
                      </div>
                      <div className="w-24 text-right text-muted-foreground font-medium">
                        <MaskedBalance amount={item.monthlyLimit} />
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <Progress
                      value={Math.min(progress, 100)}
                      indicatorColor={getProgressColor(progress)}
                      className="h-1 flex-1 bg-zinc-100 dark:bg-zinc-800"
                    />
                    <span className="text-[10px] font-black text-muted-foreground w-8 text-right italic">
                      {Math.round(progress)}%
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
