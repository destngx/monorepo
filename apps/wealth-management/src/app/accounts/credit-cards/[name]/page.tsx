'use client';

import { useState, useEffect, useMemo } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Account } from '@wealth-management/types';
import { Transaction } from '@wealth-management/types';
import { Loading } from '@/components/ui/loading';
import { Card, CardContent } from '@/components/ui/card';
import { Landmark, ArrowLeft, MoreVertical, ShieldCheck, Zap, Info, ExternalLink } from 'lucide-react';
import { UtilizationRing } from '@/features/accounts/ui/credit/utilization-ring';
import { StatementCycleBar } from '@/features/accounts/ui/credit/statement-cycle-bar';
import { DueDateCountdown } from '@/features/accounts/ui/credit/due-date-countdown';
import { MaskedBalance } from '@/components/ui/masked-balance';
import { Button } from '@/components/ui/button';
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export default function CreditCardSubPage() {
  const params = useParams();
  const router = useRouter();
  const cardName = decodeURIComponent(params.name as string);

  const [accounts, setAccounts] = useState<Account[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([
      fetch('/api/accounts').then((r) => r.json() as Promise<Account[]>),
      fetch('/api/transactions').then((r) => r.json() as Promise<Transaction[]>),
    ])
      .then(([accData, txnData]) => {
        setAccounts(accData);
        setTransactions(txnData);
        setIsLoading(false);
      })
      .catch((err) => {
        const message = getErrorMessage(err);
        const errorMsg = isAppError(err) ? err.userMessage : 'Failed to load card details. Please try again.';
        setError(errorMsg);
        console.error('Card data loading error:', message);
        setIsLoading(false);
      });
  }, []);

  const card = useMemo(() => accounts.find((a) => a.name === cardName), [accounts, cardName]);
  const cardTxns = useMemo(() => transactions.filter((t) => t.accountName === cardName), [transactions, cardName]);

  if (isLoading) return <Loading fullScreen message="Accessing vault..." />;
  if (error)
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4">
        <Landmark className="h-12 w-12 text-destructive/20" />
        <h2 className="text-xl font-bold">Error loading card</h2>
        <p className="text-sm text-muted-foreground">{error}</p>
        <Button variant="outline" onClick={() => router.back()}>
          Go Back
        </Button>
      </div>
    );
  if (!card)
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-4">
        <Landmark className="h-12 w-12 text-muted-foreground/20" />
        <h2 className="text-xl font-bold">Account not found</h2>
        <Button variant="outline" onClick={() => router.back()}>
          Go Back
        </Button>
      </div>
    );

  const balance = Math.abs(card.balance);
  const limit = card.goalAmount || 1; // Fallback to avoid division by zero
  const available = limit - balance;

  // Hypothetical logic for statement cycle based on known cards or manual config
  const isSacombank = card.name.includes('Sacombank');
  const reportDay = isSacombank ? 5 : 1;
  const dueDay = isSacombank ? 30 : 25;

  return (
    <div className="max-w-7xl mx-auto space-y-8 pb-20">
      {/* Dynamic Header */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-6 px-4">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => router.back()} className="rounded-full">
            <ArrowLeft className="h-5 w-5" />
          </Button>
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="text-[10px] font-black uppercase tracking-[0.2em] text-blue-500 bg-blue-500/10 px-2 py-0.5 rounded">
                Credit Module
              </span>
            </div>
            <h1 className="text-4xl font-black tracking-tighter flex items-center gap-3">
              {card.name}
              <ShieldCheck className="h-6 w-6 text-emerald-500" />
            </h1>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="outline" size="sm" className="gap-2 font-bold uppercase tracking-widest text-[10px]">
            <ExternalLink className="h-3 w-3" /> External Link
          </Button>
          <Button size="sm" className="gap-2 font-bold uppercase tracking-widest text-[10px]">
            Manage Card
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 px-4">
        {/* Left Column: Visual Analytics */}
        <div className="lg:col-span-2 space-y-8">
          {/* Main Card Data */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-8 bg-zinc-950 dark:bg-zinc-900/50 rounded-3xl p-8 border border-white/5 relative overflow-hidden shadow-2xl">
            <div className="absolute top-0 right-0 w-64 h-64 bg-primary/10 rounded-full blur-3xl -mr-32 -mt-32 pointer-events-none" />

            <div className="flex flex-col justify-between relative z-10">
              <div className="space-y-1">
                <p className="text-[11px] font-bold text-zinc-500 uppercase tracking-widest">Total Outstanding</p>
                <h2 className="text-5xl font-black tracking-tighter text-white">
                  <MaskedBalance amount={balance} />
                </h2>
              </div>

              <div className="grid grid-cols-2 gap-4 mt-12 bg-white/5 p-4 rounded-2xl backdrop-blur-sm">
                <div>
                  <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest mb-1">Total Limit</p>
                  <p className="text-lg font-bold tracking-tight text-white/90">
                    <MaskedBalance amount={limit} />
                  </p>
                </div>
                <div>
                  <p className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest mb-1">Available</p>
                  <p className="text-lg font-bold tracking-tight text-emerald-400">
                    <MaskedBalance amount={available} />
                  </p>
                </div>
              </div>
            </div>

            <div className="relative z-10">
              <UtilizationRing used={balance} limit={limit} />
            </div>
          </div>

          {/* Quick Insights */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card className="bg-card/30 border-dashed">
              <CardContent className="p-6">
                <div className="flex items-start gap-4">
                  <div className="p-3 bg-blue-500/10 text-blue-500 rounded-xl">
                    <Info className="h-6 w-6" />
                  </div>
                  <div className="space-y-1">
                    <h4 className="font-bold">Optimization Tip</h4>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      To maintain a healthy credit score, try keeping your utilization below 30% (
                      <MaskedBalance amount={limit * 0.3} />
                      ).
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
            <Card className="bg-card/30 border-dashed">
              <CardContent className="p-6">
                <div className="flex items-start gap-4">
                  <div className="p-3 bg-emerald-500/10 text-emerald-500 rounded-xl">
                    <Zap className="h-6 w-6" />
                  </div>
                  <div className="space-y-1">
                    <h4 className="font-bold">Rewards Tracking</h4>
                    <p className="text-xs text-muted-foreground leading-relaxed">
                      You've earned an estimated ₫120k in cashback this cycle based on transaction categories.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Transaction Summary (Static Placeholder for now) */}
          <div className="space-y-4">
            <h3 className="text-xl font-bold tracking-tight px-2">Cycle Transactions</h3>
            <div className="bg-card border rounded-2xl overflow-hidden divide-y">
              {cardTxns.slice(0, 5).map((t, i) => (
                <div key={i} className="flex items-center justify-between p-4 hover:bg-muted/50 transition-colors">
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 rounded-full bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center text-[10px] font-bold">
                      {t.payee.charAt(0)}
                    </div>
                    <div>
                      <p className="text-sm font-bold">{t.payee}</p>
                      <p className="text-[10px] text-muted-foreground uppercase tracking-widest">
                        {t.category} · {new Date(t.date).toLocaleDateString()}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm font-bold text-orange-500">
                    -<MaskedBalance amount={t.payment || 0} />
                  </p>
                </div>
              ))}
              {cardTxns.length === 0 && (
                <div className="p-12 text-center text-muted-foreground italic text-sm">
                  No transactions found for this cycle.
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Right Column: Cycle & Due Info */}
        <div className="space-y-8">
          <DueDateCountdown balance={balance} dueDay={dueDay} />

          <Card className="shadow-sm border-white/5 overflow-hidden">
            <CardContent className="p-6">
              <StatementCycleBar reportDay={reportDay} />
            </CardContent>
          </Card>

          <div className="p-6 rounded-2xl bg-muted/20 border border-white/5 space-y-4">
            <h4 className="text-sm font-bold uppercase tracking-widest text-muted-foreground">Card Settings</h4>
            <div className="space-y-2">
              {['Lock Card', 'Change PIN', 'Contact Support'].map((item) => (
                <button
                  key={item}
                  className="w-full flex items-center justify-between p-3 rounded-xl hover:bg-muted transition-colors text-sm font-medium"
                >
                  {item}
                  <MoreVertical className="h-4 w-4 opacity-40" />
                </button>
              ))}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
