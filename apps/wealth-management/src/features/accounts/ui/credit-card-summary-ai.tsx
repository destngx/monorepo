'use client';

import { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Sparkles, BrainCircuit, Lightbulb, TrendingUp } from 'lucide-react';
import { Transaction } from '@wealth-management/types';
import ReactMarkdown from 'react-markdown';
import { AIInsightRenderer } from '@wealth-management/ui';
import type { StructuredInsight } from '@wealth-management/ai/server';

import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';

interface Props {
  transactions: Transaction[];
  cardStats: any[];
}

export function CreditCardSummaryAI({ transactions, cardStats }: Props) {
  const [loading, setLoading] = useState(false);
  const [summaries, setSummaries] = useState<Record<string, StructuredInsight | string>>({});
  const [activeTab, setActiveTab] = useState('current');
  const [selectedPastMonth, setSelectedPastMonth] = useState<string>('');
  const hasGeneratedRef = useRef<Record<string, boolean>>({});

  // Generate list of previous months in the current year
  const pastMonths = (() => {
    const now = new Date();
    const currentMonth = now.getMonth();
    const currentYear = now.getFullYear();
    const months = [];
    for (let i = currentMonth - 1; i >= 0; i--) {
      const m = i + 1;
      const label = new Date(currentYear, i).toLocaleDateString('en-US', { month: 'long' });
      months.push({ value: `${String(m).padStart(2, '0')}/${currentYear}`, label: `${label} ${currentYear}` });
    }
    return months;
  })();

  // Default past month to the most recent one (last month)
  useEffect(() => {
    if (pastMonths.length > 0 && !selectedPastMonth) {
      setSelectedPastMonth(pastMonths[0].value);
    }
  }, [pastMonths, selectedPastMonth]);

  const generateSummary = async (target: string) => {
    if (loading) return;
    setLoading(true);
    try {
      const response = await fetch('/api/ai/credit-summary', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          transactions,
          cardStats,
          targetMonth: target === 'current' ? 'current' : target,
        }),
      });
      const data = await response.json();
      if (data.summary) {
        setSummaries((prev) => ({ ...prev, [target]: data.summary }));
      } else {
        throw new Error(data.error || 'Failed to generate summary');
      }
    } catch (e) {
      console.error(e);
      setSummaries((prev) => ({ ...prev, [target]: 'Failed to generate AI summary. Please try again later.' }));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    const target = activeTab === 'current' ? 'current' : selectedPastMonth;
    if (target && !hasGeneratedRef.current[target] && !summaries[target]) {
      hasGeneratedRef.current[target] = true;
      void generateSummary(target);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, selectedPastMonth]);

  const currentSummary = activeTab === 'current' ? summaries['current'] : summaries[selectedPastMonth];
  const isStructured = typeof currentSummary === 'object' && currentSummary !== null && 'sections' in currentSummary;

  return (
    <Card className="border-primary/20 bg-primary/5 shadow-lg overflow-hidden relative">
      <div className="absolute top-0 right-0 p-4 opacity-10 pointer-events-none">
        <BrainCircuit className="w-32 h-32" />
      </div>
      <CardHeader className="pb-2 border-b">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-indigo-500/10 rounded-lg">
              <Sparkles className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
            </div>
            <div>
              <CardTitle className="text-base font-bold text-indigo-900 dark:text-indigo-100 uppercase tracking-tight">
                Lộc Phát Tài Insights
              </CardTitle>
              <p className="text-[10px] text-indigo-600/70 dark:text-indigo-400/70 font-medium">
                Credit Card Optimization Summary
              </p>
            </div>
          </div>

          <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full sm:w-auto">
            <TabsList className="grid w-full grid-cols-2 h-8">
              <TabsTrigger value="current" className="text-[10px] uppercase font-bold">
                This Month
              </TabsTrigger>
              <TabsTrigger value="past" className="text-[10px] uppercase font-bold">
                Past Month
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </div>
      </CardHeader>
      <CardContent className="space-y-4 pt-4">
        {activeTab === 'past' && pastMonths.length > 0 && (
          <div className="flex items-center gap-2 mb-2 animate-in fade-in slide-in-from-top-1 duration-300">
            <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">
              Analyze Performance For:
            </span>
            <Select value={selectedPastMonth} onValueChange={setSelectedPastMonth}>
              <SelectTrigger className="w-[160px] h-7 text-[10px] font-bold bg-background/50">
                <SelectValue placeholder="Select Month" />
              </SelectTrigger>
              <SelectContent>
                {pastMonths.map((m) => (
                  <SelectItem key={m.value} value={m.value} className="text-[10px]">
                    {m.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}

        <div className="bg-indigo-50/50 dark:bg-indigo-950/20 rounded-lg p-3 border border-indigo-100/50 dark:border-indigo-900/30 min-h-[120px] flex items-center justify-center">
          {loading ? (
            <div className="flex items-center gap-2 text-xs text-muted-foreground animate-pulse py-6">
              <Sparkles className="h-4 w-4 text-indigo-400 animate-bounce" />
              <span>Analyzing {activeTab === 'current' ? 'this month' : selectedPastMonth}...</span>
            </div>
          ) : !currentSummary ? (
            <div className="flex flex-col items-center gap-3 py-4">
              <p className="text-xs text-muted-foreground text-center max-w-sm">
                Let the AI analyze your spending patterns {activeTab === 'current' ? 'this month' : 'from a past month'}{' '}
                to suggest higher cashback opportunities.
              </p>
              <Button
                onClick={() => generateSummary(activeTab === 'current' ? 'current' : selectedPastMonth)}
                disabled={loading}
                size="sm"
                className="gap-2 shadow-sm text-xs h-8"
              >
                <BrainCircuit className="h-3.5 w-3.5" />
                Analyze {activeTab === 'current' ? 'Current' : 'Selected'} Month
              </Button>
            </div>
          ) : isStructured ? (
            <div className="w-full">
              <AIInsightRenderer insight={currentSummary} />
              <div className="flex justify-end gap-2 pt-3 border-t border-indigo-100/50 dark:border-indigo-900/30 mt-3">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    const target = activeTab === 'current' ? 'current' : selectedPastMonth;
                    setSummaries((prev) => {
                      const updated = { ...prev };
                      delete updated[target];
                      return updated;
                    });
                    delete hasGeneratedRef.current[target];
                  }}
                  className="text-[10px] h-6 px-2"
                >
                  Clear
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => generateSummary(activeTab === 'current' ? 'current' : selectedPastMonth)}
                  disabled={loading}
                  className="text-[10px] h-6 px-2 gap-1 text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 hover:bg-indigo-100/50"
                >
                  <Sparkles className="h-3 w-3" />
                  Re-analyze
                </Button>
              </div>
            </div>
          ) : (
            <div className="w-full">
              <div className="prose prose-sm dark:prose-invert max-w-none text-xs leading-relaxed text-muted-foreground italic mb-4">
                <ReactMarkdown>{currentSummary as string}</ReactMarkdown>
              </div>
              <div className="flex justify-end gap-2 pt-3 border-t border-indigo-100/50 dark:border-indigo-900/30">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    const target = activeTab === 'current' ? 'current' : selectedPastMonth;
                    setSummaries((prev) => {
                      const updated = { ...prev };
                      delete updated[target];
                      return updated;
                    });
                    delete hasGeneratedRef.current[target];
                  }}
                  className="text-[10px] h-6 px-2"
                >
                  Clear
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => generateSummary(activeTab === 'current' ? 'current' : selectedPastMonth)}
                  disabled={loading}
                  className="text-[10px] h-6 px-2 gap-1 text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 hover:bg-indigo-100/50"
                >
                  <Sparkles className="h-3 w-3" />
                  Re-analyze
                </Button>
              </div>
            </div>
          )}
        </div>

        {/* Pro Tips Section */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 mt-4">
          <div className="flex gap-3 p-3 rounded-xl bg-blue-500/10 border border-blue-200/20 shadow-sm transition-all hover:shadow-md">
            <Lightbulb className="h-5 w-5 text-blue-500 shrink-0" />
            <div className="space-y-1">
              <p className="text-xs font-bold text-blue-700 dark:text-blue-400 uppercase tracking-tighter">
                Pro Tip: UNIQ
              </p>
              <p className="text-[10px] text-blue-600/80 leading-relaxed font-medium">
                Use for Grab, Fuel, and Supermarkets only. Any other use drops ROI from 20% to 0.5%.
              </p>
            </div>
          </div>
          <div className="flex gap-3 p-3 rounded-xl bg-emerald-500/10 border border-emerald-200/20 shadow-sm transition-all hover:shadow-md">
            <TrendingUp className="h-5 w-5 text-emerald-500 shrink-0" />
            <div className="space-y-1">
              <p className="text-xs font-bold text-emerald-700 dark:text-emerald-400 uppercase tracking-tighter">
                Efficiency: Platinum
              </p>
              <p className="text-[10px] text-emerald-600/80 leading-relaxed font-medium">
                Switch Shopee/Lazada transactions to this card with the 'platinum online' tag for 5% cashback.
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
