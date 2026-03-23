export const dynamic = 'force-dynamic';
export const fetchCache = 'force-no-store';

export const generateStaticParams = async () => {
  return [];
};

import { getTransactions } from "@wealth-management/services/server";
import { getAccounts } from "@wealth-management/services/server";
import { getCreditCardSummary, SACOMBANK_CASHBACK_RULES } from "@wealth-management/utils";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@wealth-management/ui";
import { MaskedBalance } from "@wealth-management/ui";
import { CreditCard, ShieldCheck, History, Info, Landmark } from 'lucide-react';
import { CreditCardSummaryAI } from '@/components/credit-cards/credit-card-summary-ai';
import { EfficiencyChart } from '@/components/credit-cards/efficiency-chart';
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@wealth-management/ui";
import { AIDataInsight } from "@/components/dashboard/ai-data-insight";
import { Transaction, Account, BankSummary, BankAccountSummary, CardStat, CardHistory } from "@wealth-management/types";

export const revalidate = 0;

export default async function CreditCardPage() {
  const [transactions, accounts]: [Transaction[], Account[]] = await Promise.all([
    getTransactions().catch(() => []),
    getAccounts().catch(() => [])
  ]);

  const banks: BankSummary[] = getCreditCardSummary(transactions, accounts);
  
  // Create a flattened list of all card stats for AI and other components if needed
  const allCardStats: CardStat[] = banks.flatMap((bank: BankSummary) => 
    bank.accounts.flatMap((acc: BankAccountSummary) => acc.cards)
  );

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
        <TabsList className="bg-muted/50 p-1">
          <TabsTrigger value="overview" className="px-6">Overview</TabsTrigger>
          <TabsTrigger value="history" className="px-6">Efficiency History</TabsTrigger>
          <TabsTrigger value="rules" className="px-6">Card Rules</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          <div className="grid grid-cols-1 gap-8">
            {banks.map((bank: BankSummary) => (
              <div key={bank.name} className="space-y-6">
                <div className="flex items-center gap-3 border-b pb-2">
                  <Landmark className="h-6 w-6 text-primary" />
                  <h2 className="text-2xl font-bold tracking-tight">{bank.name}</h2>
                </div>
                
                <div className="grid grid-cols-1  gap-6">
                  {bank.accounts.map((account: BankAccountSummary) => (
                    <Card key={account.name} className="relative overflow-hidden border-primary/10 shadow-lg bg-card/50 backdrop-blur-sm">
                      <div className="absolute top-0 right-0 w-48 h-48 -mr-24 -mt-24 rounded-full opacity-5 bg-primary" />
                      
                      <CardHeader className="pb-4 border-b bg-muted/30">
                        <div className="flex justify-between items-start">
                          <div className="space-y-1">
                            <div className="flex items-center gap-2">
                              <CreditCard className="h-5 w-5 text-primary" />
                              <CardTitle className="text-lg">{account.name}</CardTitle>
                            </div>
                            <CardDescription className="font-semibold text-foreground flex items-center gap-2">
                              Limit: <MaskedBalance amount={account.limit} />
                            </CardDescription>
                          </div>
                          <div className="text-right">
                            <p className="text-3xl font-bold tracking-tighter">{account.utilization.toFixed(1)}%</p>
                            <p className="text-[10px] text-muted-foreground uppercase font-bold">Utilization</p>
                          </div>
                        </div>
                      </CardHeader>
                      
                      <CardContent className="pt-6 space-y-8">
                        {/* Shared Progress Bar */}
                        <div className="space-y-3">
                          <div className="flex justify-between text-sm font-medium">
                            <span className="flex items-center gap-1.5">
                              <History className="h-3.5 w-3.5 text-muted-foreground" />
                              Total Spent: <MaskedBalance amount={account.totalUsage} />
                            </span>
                            <span className="text-emerald-600">Available: <MaskedBalance amount={account.remainingLimit} /></span>
                          </div>
                          <div className="w-full bg-muted rounded-full h-4 overflow-hidden shadow-inner border border-border">
                            <div
                              className={`h-full transition-all duration-1000 ease-out ${
                                account.utilization > 80 ? 'bg-orange-500' : 'bg-primary'
                              }`}
                              style={{ width: `${Math.min(account.utilization, 100)}%` }}
                            >
                              <div className="w-full h-full opacity-20 bg-[linear-gradient(45deg,rgba(255,255,255,.15)25%,transparent25%,transparent50%,rgba(255,255,255,.15)50%,rgba(255,255,255,.15)75%,transparent75%,transparent)] bg-[length:1rem_1rem] animate-[animate-stripes_1s_linear_infinite]" />
                            </div>
                          </div>
                        </div>

                        {/* Account Lifetime Summary */}
                        <div className="grid grid-cols-3 gap-2 pb-2">
                           <div className="p-2 rounded-lg bg-emerald-500/5 border border-emerald-500/10 text-center">
                              <p className="text-[9px] uppercase font-bold text-emerald-600/70">Refunds</p>
                              <p className="text-sm font-bold text-emerald-600 tracking-tighter"><MaskedBalance amount={account.totalRefund} /></p>
                           </div>
                           <div className="p-2 rounded-lg bg-orange-500/5 border border-orange-500/10 text-center">
                              <p className="text-[9px] uppercase font-bold text-orange-600/70">Card Fees</p>
                              <p className="text-sm font-bold text-orange-600 tracking-tighter">-<MaskedBalance amount={account.totalFees} /></p>
                           </div>
                           <div className="p-2 rounded-lg bg-primary/5 border border-primary/10 text-center">
                              <p className="text-[9px] uppercase font-bold text-primary/70">Total Earn</p>
                              <p className="text-sm font-bold text-primary tracking-tighter"><MaskedBalance amount={account.totalEarn} /></p>
                           </div>
                        </div>

                        {/* Individual Card Profiles within this Account */}
                        <div className="grid grid-cols-1 gap-4">
                          <p className="text-xs font-bold uppercase tracking-wider text-muted-foreground px-1">Card Profiles</p>
                          {account.cards.map((card: CardStat) => (
                            <div key={card.name} className={`p-4 rounded-xl border-l-4 shadow-sm transition-all hover:translate-x-1 duration-200 ${card.name.includes('UNIQ') ? 'border-emerald-500 bg-emerald-500/5' : 'border-blue-500 bg-blue-500/5'}`}>
                              <div className="flex justify-between items-center mb-4">
                                <h4 className="font-bold text-sm flex items-center gap-2">
                                  <div className={`w-2 h-2 rounded-full ${card.name.includes('UNIQ') ? 'bg-emerald-500' : 'bg-blue-500'}`} />
                                  {card.name.replace('Sacombank Visa ', '')}
                                </h4>
                                <div className="text-right">
                                  <div className="text-xs font-medium text-emerald-600"><MaskedBalance amount={card.estimatedCashback} /> est. cashback</div>
                                  {card.expiry && <p className="text-[9px] text-muted-foreground mt-0.5 font-mono">Expires: {card.expiry}</p>}
                                </div>
                              </div>
                              
                              <div className="grid grid-cols-3 gap-2 text-[10px]">
                                <div className="space-y-1">
                                  <p className="text-muted-foreground uppercase font-bold text-[8px]">Refund</p>
                                  <p className="font-bold tabular-nums text-emerald-600"><MaskedBalance amount={card.lifetimeCashback} /></p>
                                </div>
                                <div className="space-y-1">
                                  <p className="text-muted-foreground uppercase font-bold text-[8px]">Fees</p>
                                  <p className="font-bold tabular-nums text-orange-600">-<MaskedBalance amount={card.totalFees} /></p>
                                </div>
                                <div className="space-y-1">
                                  <p className="text-muted-foreground uppercase font-bold text-[8px]">Net Earn</p>
                                  <p className="font-bold tabular-nums text-primary">
                                    <MaskedBalance amount={card.netEarn} />
                                  </p>
                                </div>
                              </div>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </TabsContent>

        <TabsContent value="history" className="space-y-6">
           <EfficiencyChart cardStats={allCardStats} />

           <div className="flex items-start gap-2 bg-blue-500/5 text-blue-800 dark:text-blue-300 p-3 flex-row rounded-lg border border-blue-500/20 text-xs">
              <Info className="h-4 w-4 mt-0.5 shrink-0 text-blue-500" />
              <div className="space-y-1">
                 <p className="font-semibold">How Dates are Calculated</p>
                 <p className="opacity-90">
                    By default, metrics use the standard transaction date. However, for delayed <strong>Cashback Refunds</strong>, the system reads the <strong>Memo</strong> field to find the original target month (e.g., &quot;12/2025&quot;). The refund is then retroactively applied to that <em>Specific Month</em> across all charts, ensuring accurate performance tracking.
                 </p>
              </div>
           </div>

           <Card className="border-primary/5 shadow-md">
              <CardHeader>
                 <CardTitle className="text-lg">Raw Historical Data</CardTitle>
                 <CardDescription>Historical data grouped by card and month</CardDescription>
              </CardHeader>
              <CardContent>
                 <div className="relative overflow-x-auto">
                    <table className="w-full text-sm text-left">
                       <thead className="text-xs text-muted-foreground uppercase bg-muted/50">
                          <tr>
                             <th className="px-6 py-3 rounded-l-lg">Card / Month</th>
                             <th className="px-6 py-3 text-right">Spent</th>
                             <th className="px-6 py-3 text-right">Potential CB</th>
                             <th className="px-6 py-3 text-right">Actual Refund</th>
                             <th className="px-6 py-3 text-right rounded-r-lg">Efficiency (%)</th>
                          </tr>
                       </thead>
                       <tbody className="divide-y divide-border">
                          {allCardStats.map((card: CardStat) => (
                             card.history.map((h: CardHistory, i: number) => (
                                <tr key={`${card.name}-${h.month}`} className="bg-background hover:bg-muted/30 transition-colors">
                                   <td className="px-6 py-4 font-medium">
                                      {i === 0 && <span className="block text-xs font-bold text-primary mb-1 uppercase tracking-tighter">{card.name}</span>}
                                      {new Date(h.month + '-01').toLocaleDateString('en-US', { month: 'short', year: 'numeric' })}
                                   </td>
                                   <td className="px-6 py-4 text-right tabular-nums"><MaskedBalance amount={h.spent} /></td>
                                   <td className="px-6 py-4 text-right tabular-nums text-muted-foreground"><MaskedBalance amount={h.cashback} /></td>
                                   <td className="px-6 py-4 text-right tabular-nums text-emerald-600 font-bold">
                                      {h.actualRefund > 0 ? <><span className="mr-0.5">+</span><MaskedBalance amount={h.actualRefund} /></> : '-'}
                                   </td>
                                   <td className="px-6 py-4 text-right">
                                      <div className="flex items-center justify-end gap-2">
                                         <span className={`px-2 py-0.5 rounded-full text-[10px] font-bold ${
                                            h.efficiency > 4 ? 'bg-emerald-100 text-emerald-800' : h.efficiency > 1 ? 'bg-blue-100 text-blue-800' : 'bg-muted text-muted-foreground'
                                         }`}>
                                            {h.efficiency.toFixed(2)}%
                                         </span>
                                      </div>
                                   </td>
                                </tr>
                             ))
                          ))}
                       </tbody>
                    </table>
                 </div>
              </CardContent>
           </Card>
        </TabsContent>

        <TabsContent value="rules">
           <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {Object.entries(SACOMBANK_CASHBACK_RULES).map(([cardName, rules]) => (
                <Card key={cardName} className="border-primary/5 shadow-md">
                   <CardHeader className={`${cardName.includes('UNIQ') ? 'bg-emerald-500/5' : 'bg-blue-500/5'}`}>
                      <CardTitle className="text-base flex items-center gap-2">
                         <ShieldCheck className={`h-5 w-5 ${cardName.includes('UNIQ') ? 'text-emerald-500' : 'text-blue-500'}`} />
                         {cardName.split(' ').slice(2).join(' ')} Rules
                      </CardTitle>
                   </CardHeader>
                   <CardContent className="pt-6 space-y-4">
                      {rules.map((rule) => (
                        <div key={rule.tag} className={`flex justify-between items-center p-3 rounded-lg border ${rule.tag.includes('online') || rule.tag.includes('supermarket') ? 'bg-primary/5 border-primary/20' : ''}`}>
                           <div className="space-y-0.5">
                              <span className="text-sm font-semibold block">{rule.name}</span>
                              <span className="text-[10px] text-muted-foreground font-mono">tag: {rule.tag}</span>
                           </div>
                           <span className={`font-bold ${rule.rate > 1 ? 'text-emerald-600' : 'text-muted-foreground'}`}>{rule.rate}%</span>
                        </div>
                      ))}
                      <div className="p-4 rounded-lg bg-muted/30 space-y-2 text-xs text-muted-foreground">
                         <div className="flex items-center gap-2 font-medium text-foreground italic underline">Usage Policy</div>
                         <div className="flex items-center gap-2"><div className={`w-1.5 h-1.5 rounded-full ${cardName.includes('UNIQ') ? 'bg-emerald-500' : 'bg-blue-500'}`} /> Max <MaskedBalance amount={rules[0].capMonthly} /> cap for primary category</div>
                         <div className="flex items-center gap-2"><div className={`w-1.5 h-1.5 rounded-full ${cardName.includes('UNIQ') ? 'bg-emerald-500' : 'bg-blue-500'}`} /> Max <MaskedBalance amount={600000} /> total per cycle</div>
                      </div>
                   </CardContent>
                </Card>
              ))}
           </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
