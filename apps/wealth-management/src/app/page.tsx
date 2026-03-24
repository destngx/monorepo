import { getAccounts } from '@wealth-management/services/server';
import { getBudget } from '@wealth-management/services/server';
import { getTransactions } from '@wealth-management/services/server';
import { getLoans } from '@wealth-management/services/server';
import { getExchangeRate } from '@wealth-management/services/server';
import { NetworkError } from '@wealth-management/utils/errors';

import { NetWorthTrendCard } from '@/components/dashboard/net-worth-trend-card';
import { AIDailyBriefing } from '@/components/dashboard/ai-daily-briefing';
import { SnapshotCardsRow } from '@/components/dashboard/snapshot-cards-row';
import { AccountsSummary } from '@/components/dashboard/accounts-summary';
import { SpendingChart } from '@/components/dashboard/spending-chart';
import { BudgetOverview } from '@/components/dashboard/budget-overview';
import { getCategories } from '@wealth-management/services/server';
import { Account, BudgetItem, Transaction, Loan } from '@wealth-management/types';

interface CategoryWithType {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

// Setting this limits caching behavior at the page level
export const revalidate = 0;

export default async function DashboardPage() {
  // Use Promise.all to fetch everything in parallel
  const [accounts, budget, rawTransactions, loans, , categories] = await Promise.all([
    getAccounts().catch((error) => {
      throw new NetworkError('Failed to fetch accounts', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
    getBudget().catch((error) => {
      throw new NetworkError('Failed to fetch budget', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
    getTransactions().catch((error) => {
      throw new NetworkError('Failed to fetch transactions', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
    getLoans().catch((error) => {
      throw new NetworkError('Failed to fetch loans', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
    getExchangeRate().catch(() => 25400),
    getCategories().catch((error) => {
      throw new NetworkError('Failed to fetch categories', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }) as Promise<CategoryWithType[]>,
  ]);

  // Enrich transactions with categoryType for filtering in charts/stats
  const transactions = rawTransactions.map((t) => ({
    ...t,
    categoryType: categories.find((c) => c.name.toLowerCase().trim() === t.category.toLowerCase().trim())?.type,
  }));

  return (
    <div className="flex flex-col gap-8 pb-12">
      <div className="space-y-8 max-w-7xl mx-auto w-full">
        {/* Top Intelligence Section */}
        <section className="space-y-6">
          <NetWorthTrendCard accounts={accounts} transactions={transactions} loans={loans} />
          <AIDailyBriefing accounts={accounts} transactions={transactions} budget={budget} loans={loans} />
          <SnapshotCardsRow accounts={accounts} transactions={transactions} loans={loans} />
        </section>

        {/* Supporting Details */}
        <div className="grid gap-8 grid-cols-1">
          {/* Sidebar / Detailed Summaries */}
          <div className="grid grid-cols-2 gap-8">
            <AccountsSummary accounts={accounts} />
            <BudgetOverview budget={budget} transactions={transactions} />
          </div>
          {/* Main Content Area */}
          <div className=" space-y-8">
            <SpendingChart transactions={transactions} />
          </div>
        </div>
      </div>
    </div>
  );
}
