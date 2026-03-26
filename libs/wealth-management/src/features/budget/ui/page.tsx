'use client';

import { Button, Loading } from '@wealth-management/ui';
import { BudgetItem, Transaction } from '@wealth-management/types';
import { addMonths, format, subMonths } from 'date-fns';
import { useEffect, useMemo, useState } from 'react';
import { useApiErrorHandler } from '../../../hooks/use-api-error-handler';

import { AIBudgetAdvisorView } from '../ui/ai-budget-advisor-view';
import { BudgetOverviewView } from '../ui/budget-overview-view';
import { CategoryDetailView } from '../ui/category-detail-view';
import { useAISettings } from '@wealth-management/hooks';
import { cn } from '@wealth-management/utils';
import { Sparkles } from 'lucide-react';

type ViewTier = 'overview' | 'detail' | 'advisor';

export default function BudgetPage() {
  const [budgetBase, setBudgetBase] = useState<BudgetItem[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [viewDate, setViewDate] = useState(new Date());
  const [tier, setTier] = useState<ViewTier>('overview');
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);

  const [advisorData, setAdvisorData] = useState<any>(null);
  const [fetchingAdvisor, setFetchingAdvisor] = useState(false);
  const { settings, mounted } = useAISettings();
  const { withErrorHandling } = useApiErrorHandler();

  useEffect(() => {
    Promise.all([
      withErrorHandling(async () => {
        const res = await fetch('/api/budget');
        if (!res.ok) throw new Error(`Failed to load budget: ${res.statusText}`);
        return res.json();
      }, 'Failed to load budget'),
      withErrorHandling(async () => {
        const res = await fetch('/api/transactions');
        if (!res.ok) throw new Error(`Failed to load transactions: ${res.statusText}`);
        return res.json();
      }, 'Failed to load transactions'),
    ])
      .then(([budgetData, txData]) => {
        if (budgetData) setBudgetBase(budgetData);
        if (txData) setTransactions(txData);
        setLoading(false);
      })
      .catch((e) => {
        console.error(e);
        setLoading(false);
      });
  }, [withErrorHandling]);

  const fetchAdvisor = async () => {
    if (!mounted) return;
    setFetchingAdvisor(true);
    try {
      const res = await withErrorHandling(async () => {
        const response = await fetch('/api/ai/budget-advisor', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            budget: budgetBase,
            transactions,
            date: format(viewDate, 'yyyy-MM-dd'),
            modelId: settings.modelId,
          }),
        });
        if (!response.ok) throw new Error(`Failed to get budget advisor: ${response.statusText}`);
        return response;
      }, 'Failed to get budget advisor');
      if (res && res.ok) {
        const data = await res.json();
        setAdvisorData(data);
      }
    } catch (err) {
      console.error('Advisor Fetch Error:', err);
    } finally {
      setFetchingAdvisor(false);
    }
  };

  useEffect(() => {
    if (budgetBase.length > 0 && transactions.length > 0 && mounted) {
      setAdvisorData(null); // Reset to show skeletons
      fetchAdvisor();
    }
  }, [viewDate, mounted, budgetBase.length, transactions.length]);

  const handlePrev = () => setViewDate((prev) => subMonths(prev, 1));
  const handleNext = () => setViewDate((prev) => addMonths(prev, 1));
  const handleReset = () => setViewDate(new Date());

  // Dynamically calculate the budget vs actuals
  const budget = useMemo(() => {
    const activeYear = viewDate.getFullYear();
    const activeMonth = viewDate.getMonth();
    const monthKey = `${activeYear}-${String(activeMonth + 1).padStart(2, '0')}`;

    return budgetBase.map((b) => {
      const monthTxns = transactions.filter((t) => {
        if (t.category !== b.category) return false;
        const txDate = new Date(t.date);
        return txDate.getMonth() === activeMonth && txDate.getFullYear() === activeYear;
      });

      const monthlySpent = Math.max(
        0,
        monthTxns.reduce((sum, t) => sum + (t.payment || 0) - (t.deposit || 0), 0),
      );
      const exactMonthlyLimit = b.monthlyLimits?.[monthKey] ?? b.monthlyLimit;

      return {
        ...b,
        monthlyLimit: exactMonthlyLimit,
        monthlySpent,
        monthlyRemaining: Math.max(0, exactMonthlyLimit - monthlySpent),
      };
    });
  }, [budgetBase, transactions, viewDate]);

  if (loading) return <Loading fullScreen message="Thinking..." />;

  const activeBudgets = budget.filter((b) => b.monthlyLimit > 0 || b.monthlySpent > 0);
  const currentCategoryData = selectedCategory ? budget.find((b) => b.category === selectedCategory) : null;
  const currentCategoryTxns = selectedCategory
    ? transactions.filter((t) => {
        if (t.category !== selectedCategory) return false;
        const txDate = new Date(t.date);
        return txDate.getMonth() === viewDate.getMonth() && txDate.getFullYear() === viewDate.getFullYear();
      })
    : [];

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      {/* Module Navigation */}
      <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Adaptive Budgeter</h2>
          <p className="text-muted-foreground text-sm font-medium">
            From static limits to AI-driven behavior coaching.
          </p>
        </div>

        <div className="flex rounded-lg border bg-muted p-1 gap-1">
          <Button
            variant={tier === 'overview' ? 'secondary' : 'ghost'}
            size="sm"
            onClick={() => setTier('overview')}
            className={cn(
              'text-xs font-bold px-6 transition-all duration-300',
              tier === 'overview'
                ? 'bg-background shadow-sm text-primary'
                : 'text-muted-foreground hover:text-foreground',
            )}
          >
            Overview
          </Button>
          <Button
            variant={tier === 'advisor' ? 'secondary' : 'ghost'}
            size="sm"
            onClick={() => setTier('advisor')}
            className={cn(
              'text-xs font-bold px-6 gap-2 transition-all duration-500 relative overflow-hidden group border-0',
              tier === 'advisor'
                ? 'bg-indigo-600 text-white shadow-[0_0_25px_rgba(79,70,229,0.3)]'
                : 'text-muted-foreground hover:text-indigo-600 hover:bg-zinc-100 dark:hover:bg-zinc-800',
            )}
          >
            {/* Shimmer Border Effect */}
            <div className="absolute inset-x-0 inset-y-0 p-[1px] pointer-events-none rounded-[inherit]">
              <div className="absolute inset-[-1000%] animate-shine bg-[conic-gradient(from_90deg_at_50%_50%,#4f46e5_0%,#818cf8_25%,#4f46e5_50%,#818cf8_75%,#4f46e5_100%)] opacity-40 group-hover:opacity-80 transition-opacity" />
              <div className="absolute inset-0 bg-muted rounded-[inherit] shine-mask" />
              {tier === 'advisor' && <div className="absolute inset-0 bg-indigo-600 rounded-[inherit] shine-mask" />}
            </div>

            <Sparkles
              className={cn('h-3.5 w-3.5 relative z-10', tier === 'advisor' ? 'text-indigo-200' : 'text-indigo-500')}
            />
            <span className="relative z-10">AI Advisor</span>
          </Button>
        </div>
      </div>

      {tier === 'overview' && (
        <BudgetOverviewView
          budget={activeBudgets}
          date={viewDate}
          onPrev={handlePrev}
          onNext={handleNext}
          onReset={handleReset}
          onSelectCategory={(c) => {
            setSelectedCategory(c);
            setTier('detail');
          }}
          briefing={advisorData?.briefing}
          detailedBrief={advisorData?.detailedBrief}
          insights={advisorData?.summaryInsights}
          isLoadingAdvisor={fetchingAdvisor}
        />
      )}

      {tier === 'detail' && selectedCategory && (
        <CategoryDetailView
          category={selectedCategory}
          limit={currentCategoryData?.monthlyLimit || 0}
          spent={currentCategoryData?.monthlySpent || 0}
          transactions={currentCategoryTxns}
          onBack={() => setTier('overview')}
          onAdjustLimit={(val) => {
            console.log('Adjust limit to', val);
            // In real app, call API to update sheet
          }}
        />
      )}

      {tier === 'advisor' && <AIBudgetAdvisorView data={advisorData} />}
    </div>
  );
}
