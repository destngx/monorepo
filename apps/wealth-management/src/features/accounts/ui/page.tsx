'use client';

import { useState, useEffect, useMemo } from 'react';
import Link from 'next/link';
import { MaskedBalance } from '@wealth-management/ui/masked-balance';
import { Account } from '../model/types';
import { Transaction } from '@wealth-management/types';
import { ACCOUNT_TYPES } from '@wealth-management/features/accounts/model/types';
import { type AccountType } from '../model/types';
import { Card, CardContent } from '@wealth-management/ui/card';
import { Progress } from '@wealth-management/ui/progress';
import {
  Landmark,
  Wallet,
  Coins,
  Archive,
  CreditCard,
  BarChart3,
  ArrowUpRight,
  Target,
  Info,
  Zap,
  ChevronDown,
  ChevronRight,
  Flag,
  Bitcoin,
  DollarSign,
  TrendingUp,
} from 'lucide-react';
import { Loading } from '@wealth-management/ui/loading';
import { AccountReviewAI } from './account-review-ai';
import { AIDataInsight } from '@/components/dashboard/ai-data-insight';
import { AccountTrendSparkline } from './account-trend-sparkline';

function getIconForAccountType(iconName: string): React.ReactNode {
  const iconMap: Record<string, React.ReactNode> = {
    wallet: <Wallet className="h-4 w-4" />,
    inbox: <Coins className="h-4 w-4" />,
    'piggy-bank': <Landmark className="h-4 w-4" />,
    archive: <Archive className="h-4 w-4" />,
    'trending-down': <TrendingUp className="h-4 w-4" />,
    bank: <Landmark className="h-4 w-4" />,
    bitcoin: <Bitcoin className="h-4 w-4" />,
    'dollar-sign': <DollarSign className="h-4 w-4" />,
    'trending-up': <BarChart3 className="h-4 w-4" />,
  };
  return iconMap[iconName] || <Wallet className="h-4 w-4" />;
}

const TYPE_CONFIG: Record<AccountType, { label: string; colorTitle: string; bgSoft: string }> = {
  'active use': {
    label: 'Everyday Banking',
    colorTitle: 'text-emerald-600 dark:text-emerald-500',
    bgSoft: 'bg-emerald-500/10',
  },
  'rarely use': {
    label: 'Savings & Reserves',
    colorTitle: 'text-blue-600 dark:text-blue-500',
    bgSoft: 'bg-blue-500/10',
  },
  'long holding': {
    label: 'Investments',
    colorTitle: 'text-indigo-600 dark:text-indigo-500',
    bgSoft: 'bg-indigo-500/10',
  },
  'negative active use': {
    label: 'Credit Cards & Loans',
    colorTitle: 'text-slate-800 dark:text-slate-200',
    bgSoft: 'bg-slate-500/10',
  },
  deprecated: {
    label: 'Archived Accounts',
    colorTitle: 'text-gray-500',
    bgSoft: 'bg-gray-500/10',
  },
  bank: {
    label: 'Bank Accounts',
    colorTitle: 'text-indigo-600 dark:text-indigo-500',
    bgSoft: 'bg-indigo-500/10',
  },
  crypto: {
    label: 'Crypto',
    colorTitle: 'text-yellow-600 dark:text-yellow-500',
    bgSoft: 'bg-yellow-500/10',
  },
  cash: {
    label: 'Cash',
    colorTitle: 'text-green-600 dark:text-green-500',
    bgSoft: 'bg-green-500/10',
  },
  investment: {
    label: 'Investment',
    colorTitle: 'text-purple-600 dark:text-purple-500',
    bgSoft: 'bg-purple-500/10',
  },
};

const TYPE_ORDER: AccountType[] = ['active use', 'long holding', 'rarely use', 'negative active use', 'deprecated'];

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(new Set(['active use', 'negative active use']));

  useEffect(() => {
    void Promise.all([
      fetch('/api/accounts').then((r) => r.json() as Promise<Account[]>),
      fetch('/api/transactions').then((r) => r.json() as Promise<Transaction[]>),
    ])
      .then(([accData, txnData]) => {
        setAccounts(accData);
        setTransactions(txnData);
        setIsLoading(false);
      })
      .catch((e: unknown) => {
        console.error('Failed to fetch account/transaction data', e);
        setIsLoading(false);
      });
  }, []);

  const toggleSection = (type: string) => {
    setExpandedSections((prev) => {
      const next = new Set(prev);
      if (next.has(type)) next.delete(type);
      else next.add(type);
      return next;
    });
  };

  const { grouped, sortedTypes, totalAssets, totalLiabilities, totalNetWorth } = useMemo(() => {
    const grouped = accounts.reduce<Record<string, Account[]>>((map, acc) => {
      const key = acc.type || 'bank';
      if (!map[key]) map[key] = [];
      map[key].push(acc);
      return map;
    }, {});

    const sortedTypes = TYPE_ORDER.filter((t) => grouped[t]?.length > 0);
    Object.keys(grouped).forEach((t) => {
      if (!sortedTypes.includes(t as AccountType)) sortedTypes.push(t as AccountType);
    });

    const totalAssets = accounts.filter((a) => a.balance > 0).reduce((sum: number, a) => sum + a.balance, 0);
    const totalLiabilities = Math.abs(
      accounts.filter((a) => a.balance < 0).reduce((sum: number, a) => sum + a.balance, 0),
    );
    const totalNetWorth = totalAssets - totalLiabilities;

    return { grouped, sortedTypes, totalAssets, totalLiabilities, totalNetWorth };
  }, [accounts]);

  return (
    <div className="space-y-8 max-w-7xl mx-auto pb-10">
      {/* Header Section */}
      {/* The header title and description were removed as per instruction */}

      {isLoading ? (
        <Loading fullScreen message="Thinking..." />
      ) : accounts.length === 0 ? (
        <div className="col-span-full py-20 flex flex-col items-center justify-center text-muted-foreground border border-dashed rounded-xl bg-card/30">
          <Wallet className="h-12 w-12 mb-4 opacity-20" />
          <h3 className="text-lg font-medium text-foreground">No accounts found</h3>
          <p className="text-sm">Connect your Google Sheet database to import accounts.</p>
        </div>
      ) : (
        <>
          {/* Wealth Summary Cards */}
          <div className="grid gap-2 grid-cols-1 md:grid-cols-3">
            <Card className="bg-gradient-to-br from-card to-card border shadow-sm">
              <CardContent className="px-3 py-2 flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <div className="w-1.5 h-1.5 rounded-full bg-emerald-500"></div>
                  <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground leading-none mt-0.5">
                    Total Assets
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-base font-bold tracking-tight text-emerald-600 dark:text-emerald-500">
                    <MaskedBalance amount={totalAssets} />
                  </span>
                  <AIDataInsight
                    type="Total Assets"
                    description="Aggregate value of all accounts with positive balances, including cash, savings, and investments."
                    data={accounts.filter((a) => a.balance > 0)}
                  />
                </div>
              </CardContent>
            </Card>
            <Card className="bg-gradient-to-br from-card to-card border shadow-sm">
              <CardContent className="px-3 py-2 flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <div className="w-1.5 h-1.5 rounded-full bg-orange-500"></div>
                  <span className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground leading-none mt-0.5">
                    Total Liabilities
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-base font-bold tracking-tight text-orange-500 dark:text-orange-400">
                    <MaskedBalance amount={totalLiabilities} />
                  </span>
                  <AIDataInsight
                    type="Total Liabilities"
                    description="Aggregate value of all accounts with negative balances, representing credit card debt and loans."
                    data={accounts.filter((a) => a.balance < 0)}
                  />
                </div>
              </CardContent>
            </Card>
            <Card className="bg-slate-900 border-none shadow-sm dark:border dark:border-border/50">
              <CardContent className="px-3 py-2 flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <ArrowUpRight className="h-3 w-3 text-emerald-400" />
                  <span className="text-[10px] font-medium uppercase tracking-wider text-slate-300 leading-none mt-0.5">
                    Net Worth
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <span className="text-base font-bold tracking-tight text-white">
                    <MaskedBalance amount={totalNetWorth} />
                  </span>
                  <AIDataInsight
                    type="Net Worth"
                    description="Total liquid wealth calculated as Total Assets minus Total Liabilities."
                    data={{ totalAssets, totalLiabilities, totalNetWorth }}
                  />
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Module Quick Access */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Link href="/accounts/credit-cards">
              <Card className="group hover:border-violet-500/50 transition-all duration-300 cursor-pointer bg-gradient-to-br from-violet-500/5 to-transparent border-violet-500/10 shadow-sm overflow-hidden relative">
                <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                  <CreditCard className="w-16 h-16 text-violet-500" />
                </div>
                <CardContent className="p-5 flex items-center justify-between relative z-10">
                  <div className="flex items-center gap-4">
                    <div className="p-3 bg-violet-500 text-white rounded-xl shadow-lg shadow-violet-500/20">
                      <CreditCard className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="text-base font-bold tracking-tight">Credit Intelligence</h3>
                      <p className="text-xs text-muted-foreground">Manage limits, rewards and optimization</p>
                    </div>
                  </div>
                  <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-violet-500 group-hover:translate-x-1 transition-all" />
                </CardContent>
              </Card>
            </Link>
            <Link href="/accounts/loans">
              <Card className="group hover:border-amber-500/50 transition-all duration-300 cursor-pointer bg-gradient-to-br from-amber-500/5 to-transparent border-amber-500/10 shadow-sm overflow-hidden relative">
                <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                  <Target className="w-16 h-16 text-amber-500" />
                </div>
                <CardContent className="p-5 flex items-center justify-between relative z-10">
                  <div className="flex items-center gap-4">
                    <div className="p-3 bg-amber-500 text-white rounded-xl shadow-lg shadow-amber-500/20">
                      <Target className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="text-base font-bold tracking-tight">Debt Tracking</h3>
                      <p className="text-xs text-muted-foreground">Monitor loans, goals and paydown progress</p>
                    </div>
                  </div>
                  <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-amber-500 group-hover:translate-x-1 transition-all" />
                </CardContent>
              </Card>
            </Link>
            <Link href="/accounts/goals">
              <Card className="group hover:border-emerald-500/50 transition-all duration-300 cursor-pointer bg-gradient-to-br from-emerald-500/5 to-transparent border-emerald-500/10 shadow-sm overflow-hidden relative">
                <div className="absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity">
                  <Flag className="w-16 h-16 text-emerald-500" />
                </div>
                <CardContent className="p-5 flex items-center justify-between relative z-10">
                  <div className="flex items-center gap-4">
                    <div className="p-3 bg-emerald-500 text-white rounded-xl shadow-lg shadow-emerald-500/20">
                      <Flag className="h-5 w-5" />
                    </div>
                    <div>
                      <h3 className="text-base font-bold tracking-tight">Financial Goals</h3>
                      <p className="text-xs text-muted-foreground">Track progress toward your milestones</p>
                    </div>
                  </div>
                  <ChevronRight className="h-5 w-5 text-muted-foreground group-hover:text-emerald-500 group-hover:translate-x-1 transition-all" />
                </CardContent>
              </Card>
            </Link>
          </div>

          {/* AI Review Section */}
          <AccountReviewAI
            accounts={accounts}
            totalAssets={totalAssets}
            totalLiabilities={totalLiabilities}
            totalNetWorth={totalNetWorth}
          />

          {/* High Density Table Layout */}
          <div className="space-y-0 bg-card rounded-xl border shadow-sm overflow-hidden">
            {sortedTypes.map((type, index) => {
              const config = TYPE_CONFIG[type] || {
                label: type,
                icon: <Coins className="h-4 w-4" />,
                colorTitle: 'text-gray-500',
                bgSoft: 'bg-muted',
              };
              const typeAccounts = grouped[type];
              const subtotal = typeAccounts.reduce((s, a) => s + a.balance, 0);
              const isCredit = type === 'negative active use';

              return (
                <div
                  key={type}
                  className={`animate-in fade-in duration-500 fill-mode-both ${index > 0 ? 'border-t' : ''}`}
                  style={{ animationDelay: `${TYPE_ORDER.indexOf(type) * 100}ms` }}
                >
                  {/* Category Header Row */}
                  <div
                    className="flex items-center justify-between px-6 py-4 bg-muted/30 cursor-pointer hover:bg-muted/50 transition-colors"
                    onClick={() => toggleSection(type)}
                  >
                    <div className="flex items-center gap-3">
                      <div className="text-muted-foreground mr-1">
                        {expandedSections.has(type) ? (
                          <ChevronDown className="h-4 w-4" />
                        ) : (
                          <ChevronRight className="h-4 w-4" />
                        )}
                      </div>
                      <div className={`p-1.5 rounded-md ${config.bgSoft} ${config.colorTitle}`}>
                        {getIconForAccountType(ACCOUNT_TYPES[type].icon)}
                      </div>
                      <h3 className="text-sm font-semibold uppercase tracking-wider">{config.label}</h3>
                      <AIDataInsight
                        type={`${config.label} Group`}
                        description={`Summary of accounts in the ${config.label} category, showing balances and goals.`}
                        data={typeAccounts}
                      />
                    </div>
                    {type !== 'deprecated' && (
                      <span className={`text-sm font-bold tracking-tight ${subtotal >= 0 ? '' : 'text-orange-500'}`}>
                        <MaskedBalance amount={Math.abs(subtotal)} />
                      </span>
                    )}
                  </div>

                  {/* Dense List */}
                  {expandedSections.has(type) && (
                    <div className="divide-y divide-border/50">
                      {typeAccounts.map((acc, idx) => {
                        const isCc = isCredit;

                        return (
                          <div
                            key={idx}
                            className="flex flex-col sm:flex-row sm:items-center justify-between px-6 py-4 hover:bg-muted/10 transition-colors gap-4 relative"
                          >
                            {/* Left: Name & Note */}
                            <div className="flex-1 min-w-[200px] flex items-center gap-4">
                              <AccountTrendSparkline
                                accountName={acc.name}
                                currentBalance={acc.balance}
                                transactions={transactions}
                              />
                              <div>
                                <h4 className={`font-medium ${isCc ? 'text-foreground' : ''}`}>
                                  {isCc ? (
                                    <Link
                                      href={`/accounts/credit-cards/${encodeURIComponent(acc.name)}`}
                                      className="hover:underline text-blue-600 dark:text-blue-400 flex items-center gap-1"
                                    >
                                      {acc.name}
                                      <ChevronRight className="h-3 w-3" />
                                    </Link>
                                  ) : acc.type === 'negative active use' ||
                                    (acc.note && acc.note.toLowerCase().includes('loan')) ? (
                                    <Link
                                      href={`/accounts/loans`}
                                      className="hover:underline text-amber-600 dark:text-amber-400 flex items-center gap-1"
                                    >
                                      {acc.name}
                                      <ChevronRight className="h-3 w-3" />
                                    </Link>
                                  ) : (
                                    acc.name
                                  )}
                                </h4>
                                {acc.note && !isCc && (
                                  <p className="text-[13px] text-muted-foreground mt-0.5 max-w-sm">
                                    {acc.note}
                                  </p>
                                )}
                                {isCc && acc.name.includes('Sacombank') && (
                                  <div className="mt-1 space-y-0.5">
                                    <p className="text-[10px] text-blue-600 dark:text-blue-400 flex items-center gap-1 font-medium">
                                      <Info className="h-2.5 w-2.5" /> Report: 5th of month
                                    </p>
                                    <p className="text-[10px] text-emerald-600 dark:text-emerald-400 flex items-center gap-1 font-medium">
                                      <Zap className="h-2.5 w-2.5" /> Pay before: End of month
                                    </p>
                                  </div>
                                )}
                                {!isCc && acc.dueDate && (
                                  <p className="text-xs text-amber-600 dark:text-amber-500 mt-0.5 flex items-center gap-1">
                                    <Target className="h-3 w-3" /> Target: {acc.dueDate}
                                  </p>
                                )}
                              </div>
                            </div>

                            {/* Middle: Progress Bar (Credit Cards & Goals) */}
                            <div className="w-full sm:w-1/3 flex-shrink-0">
                              {isCc && acc.goalAmount !== null ? (
                                <div className="space-y-1.5 w-full pt-1">
                                  <div className="flex justify-between text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
                                    <span>{Math.round((Math.abs(acc.balance) / acc.goalAmount) * 100)}% Used</span>
                                    <span>
                                      Limit: <MaskedBalance amount={acc.goalAmount} />
                                    </span>
                                  </div>
                                  <Progress
                                    value={Math.min(100, (Math.abs(acc.balance) / acc.goalAmount) * 100)}
                                    className="h-1.5"
                                    indicatorColor={
                                      Math.abs(acc.balance) / acc.goalAmount > 0.8
                                        ? 'bg-orange-500'
                                        : 'bg-slate-700 dark:bg-slate-300'
                                    }
                                  />
                                  {acc.note && (
                                    <p className="text-[11px] text-emerald-600 dark:text-emerald-500 text-right mt-1 font-medium">
                                      Avail:{' '}
                                      {/^\d+/.test(acc.note) ? (
                                        <MaskedBalance amount={parseFloat(acc.note)} />
                                      ) : (
                                        acc.note
                                      )}
                                    </p>
                                  )}
                                </div>
                              ) : !isCc && acc.goalAmount !== null && acc.goalProgress !== null ? (
                                <div className="space-y-1.5 w-full pt-1">
                                  <div className="flex justify-between text-[11px] font-medium text-muted-foreground uppercase tracking-wider">
                                    <span>
                                      Goal: <MaskedBalance amount={acc.goalAmount} />
                                    </span>
                                    <span className={config.colorTitle}>{Math.round(acc.goalProgress)}%</span>
                                  </div>
                                  <Progress value={Math.max(0, Math.min(100, acc.goalProgress))} className="h-1.5" />
                                </div>
                              ) : null}
                            </div>

                            {/* Right: Balance */}
                            <div className="sm:w-32 sm:text-right flex-shrink-0">
                              <div
                                className={`text-[15px] font-bold tracking-tight ${acc.balance < 0 ? 'text-orange-500' : 'text-foreground'}`}
                              >
                                <MaskedBalance amount={Math.abs(acc.balance)} />
                              </div>
                              {!isCc && acc.clearedBalance !== acc.balance && (
                                <p className="text-[11px] text-muted-foreground uppercase tracking-wider mt-0.5">
                                  Clear: <MaskedBalance amount={acc.clearedBalance} />
                                </p>
                              )}
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </>
      )}
    </div>
  );
}
