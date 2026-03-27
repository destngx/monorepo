import { getAccounts } from '@wealth-management/services/server';
import { getBudget } from '@wealth-management/services/server';
import { getTransactions } from '@wealth-management/services/server';
import { getLoans } from '@wealth-management/services/server';
import { getExchangeRate } from '@wealth-management/services/server';
import { getCategories } from '@wealth-management/services/server';

import { NetWorthTrendCard } from '@/components/dashboard/net-worth-trend-card';
import { AIDailyBriefing } from '@/components/dashboard/ai-daily-briefing';
import { SnapshotCardsRow } from '@/components/dashboard/snapshot-cards-row';
import { AccountsSummary } from '@/components/dashboard/accounts-summary';
import { SpendingChart } from '@/components/dashboard/spending-chart';
import { BudgetOverview } from '@/components/dashboard/budget-overview';
import { ServerErrorNotifier } from '@/components/server-error-notifier';
import { Account, BudgetItem, Transaction, Loan } from '@wealth-management/types';
import { getUserMessage } from '@wealth-management/utils/errors';

interface CategoryWithType {
  name: string;
  type: 'income' | 'expense' | 'non-budget';
}

// Setting this limits caching behavior at the page level
export const revalidate = 0;

export default async function DashboardPage() {
  const serverErrors: string[] = [];

  const [accounts, budget, rawTransactions, loans, , categories] = await Promise.all([
    getAccounts().catch((error) => {
      console.error('Failed to fetch accounts:', error);
      serverErrors.push(`Accounts: ${getUserMessage(error)}`);
      return [];
    }),
    getBudget().catch((error) => {
      console.error('Failed to fetch budget:', error);
      serverErrors.push(`Budget: ${getUserMessage(error)}`);
      return [];
    }),
    getTransactions().catch((error) => {
      console.error('Failed to fetch transactions:', error);
      serverErrors.push(`Transactions: ${getUserMessage(error)}`);
      return [];
    }),
    getLoans().catch((error) => {
      console.error('Failed to fetch loans:', error);
      serverErrors.push(`Loans: ${getUserMessage(error)}`);
      return [];
    }),
    getExchangeRate().catch(() => 25400),
    getCategories().catch((error) => {
      console.error('Failed to fetch categories:', error);
      serverErrors.push(`Categories: ${getUserMessage(error)}`);
      return [];
    }) as Promise<CategoryWithType[]>,
  ]);

  // Enrich transactions with categoryType for filtering in charts/stats
  const transactions = rawTransactions.map((t) => ({
    ...t,
    categoryType: categories.find((c) => c.name.toLowerCase().trim() === t.category.toLowerCase().trim())?.type,
  }));

  return (
    <div className="flex flex-col gap-8 pb-12 min-h-screen bg-dashboard-gradient">
      <ServerErrorNotifier errors={serverErrors} />
      <div className="space-y-8 max-w-7xl mx-auto w-full px-4 sm:px-6">
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
