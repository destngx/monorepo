export const dynamic = 'force-dynamic';
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

import { getTransactions } from '@wealth-management/services/server';
import { getAccounts } from '@wealth-management/services/server';
import { getCreditCardSummary } from '@wealth-management/utils';
import { NetworkError } from '@wealth-management/utils/errors';
import { CreditCardSummaryAI } from '@/components/credit-cards/credit-card-summary-ai';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@wealth-management/ui';
import { AIDataInsight } from '@wealth-management/ui';
import { Transaction, Account, BankSummary, CardStat } from '@wealth-management/types';
import { CreditCardOverview } from '@/features/accounts/ui/credit/CreditCardOverview';
import { CreditCardHistory } from '@/features/accounts/ui/credit/CreditCardHistory';
import { CreditCardRules } from '@/features/accounts/ui/credit/CreditCardRules';

export const revalidate = 0;

export default async function CreditCardPage() {
  const [transactions, accounts]: [Transaction[], Account[]] = await Promise.all([
    getTransactions().catch((error) => {
      throw new NetworkError('Failed to fetch transactions', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
    getAccounts().catch((error) => {
      throw new NetworkError('Failed to fetch accounts', {
        context: { original: error instanceof Error ? error.message : String(error) },
      });
    }),
  ]);

  const banks: BankSummary[] = getCreditCardSummary(transactions, accounts);

  // Create a flattened list of all card stats for AI and other components if needed
  const allCardStats: CardStat[] = banks.flatMap((bank: BankSummary) => bank.accounts.flatMap((acc: any) => acc.cards));

  return (
    <div className="container mx-auto py-8 px-4 space-y-8 max-w-6xl">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2 text-zinc-900 dark:text-zinc-100">
          <h1 className="text-3xl font-bold tracking-tight">Credit Card</h1>
          <AIDataInsight
            type="Credit Card Analysis"
            description="Overall analysis of credit card utilization and efficiency."
            data={allCardStats}
          />
        </div>
        <p className="text-muted-foreground italic text-sm">
          Maximize your cashback by using the right card for the right purchase.
        </p>
      </div>

      {/* AI Summary Section */}
      <CreditCardSummaryAI transactions={transactions} cardStats={allCardStats} />

      <Tabs defaultValue="overview" className="space-y-6">
        <div className="overflow-x-auto pb-2 scrollbar-hide">
          <TabsList className="bg-muted/50 p-1 min-w-max">
            <TabsTrigger value="overview" className="px-6">
              Overview
            </TabsTrigger>
            <TabsTrigger value="history" className="px-6">
              Efficiency History
            </TabsTrigger>
            <TabsTrigger value="rules" className="px-6">
              Card Rules
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="overview" className="space-y-6">
          <CreditCardOverview banks={banks} />
        </TabsContent>

        <TabsContent value="history" className="space-y-6">
          <CreditCardHistory allCardStats={allCardStats} />
        </TabsContent>

        <TabsContent value="rules">
          <CreditCardRules />
        </TabsContent>
      </Tabs>
    </div>
  );
}
