"use client";

import { useEffect, useState } from "react";
import { Card, CardContent } from "@/components/ui/card";
import { Sparkles, Loader2, Info, AlertCircle, AlertTriangle } from "lucide-react";
import { Account } from "@wealth-management/types";
import { Transaction } from "@wealth-management/types";
import { BudgetItem } from "@wealth-management/types";
import { Loan } from "@wealth-management/types";
import { AIInsightPatterns } from "./ai-insight-patterns";
import { useAISettings } from "@/hooks/use-ai-settings";

interface AIDailyBriefingProps {
  accounts: Account[];
  transactions: Transaction[];
  budget: BudgetItem[];
  loans: Loan[];
}

interface Alert {
  type: 'critical' | 'warning' | 'info';
  title: string;
  message: string;
}

export function AIDailyBriefing({ accounts, transactions, budget, loans }: AIDailyBriefingProps) {
  const [data, setData] = useState<{ briefing: string; alerts: Alert[]; patterns?: any } | null>(null);
  const [loading, setLoading] = useState(true);
  const { settings, mounted } = useAISettings();

  useEffect(() => {
    if (!mounted) return;

    async function fetchBriefing() {
      setLoading(true);
      try {
        const res = await fetch('/api/ai/intelligence-briefing', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ 
            accounts, 
            transactions, 
            budget, 
            loans,
            modelId: settings.modelId 
          }),
        });
        if (res.ok) {
          const json = await res.json();
          setData(json);
        }
      } catch (err) {
        console.error("Failed to fetch AI briefing", err);
      } finally {
        setLoading(false);
      }
    }

    fetchBriefing();
  }, [accounts, transactions, budget, loans, settings.modelId, mounted]);

  if (loading) {
    return (
      <Card className="border-none bg-zinc-50 dark:bg-zinc-900/50 animate-pulse">
        <CardContent className="p-6 flex items-center justify-center gap-3">
          <Loader2 className="h-4 w-4 animate-spin text-primary" />
          <span className="text-sm font-medium text-muted-foreground italic">AI Analyst is calculating your daily briefing...</span>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Briefing Card */}
      <Card className="border-none bg-white dark:bg-zinc-950 shadow-xl shadow-zinc-200/50 dark:shadow-none overflow-hidden relative">
        <div className="absolute top-0 left-0 w-1 h-full bg-primary" />
        <CardContent className="p-6">
          <div className="flex items-start gap-4">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Sparkles className="h-5 w-5 text-primary" />
            </div>
            <div className="space-y-1">
              <h3 className="text-sm font-bold uppercase tracking-wider text-primary flex items-center gap-2">
                Executive Briefing
              </h3>
              <p className="text-zinc-600 dark:text-zinc-400 text-sm leading-relaxed font-medium">
                {data?.briefing || "Your financial data is up to date. Review the patterns below for deeper analysis."}
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Advanced Patterns */}
      <AIInsightPatterns patterns={data?.patterns} />

      {/* Alerts Strip */}
      {data?.alerts && data.alerts.length > 0 && (
        <>  
        {/* Alerts title here */}
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-bold uppercase tracking-wider text-primary flex items-center gap-2">
              Alerts
            </h3>
          </div>
        <div className="flex gap-3 overflow-x-auto pb-2 scrollbar-none">
          {data.alerts.map((alert, i) => (
            <div 
              key={i}
              className={`flex-none w-[280px] p-4 rounded-xl border flex gap-3 transition-all hover:scale-[1.02] cursor-default ${
                alert.type === 'critical' ? 'bg-rose-50 border-rose-100 text-rose-900 dark:bg-rose-950/20 dark:border-rose-900/50 dark:text-rose-200' :
                alert.type === 'warning' ? 'bg-amber-50 border-amber-100 text-amber-900 dark:bg-amber-950/20 dark:border-amber-900/50 dark:text-amber-200' :
                'bg-blue-50 border-blue-100 text-blue-900 dark:bg-blue-950/20 dark:border-blue-900/50 dark:text-blue-200'
              }`}
            >
              <div className="shrink-0 pt-1">
                {alert.type === 'critical' ? <AlertCircle className="h-4 w-4 text-rose-500" /> :
                 alert.type === 'warning' ? <AlertTriangle className="h-4 w-4 text-amber-500" /> :
                 <Info className="h-4 w-4 text-blue-500" />}
              </div>
              <div className="space-y-1">
                <h4 className="text-xs font-bold uppercase tracking-wide">{alert.title}</h4>
                <p className="text-xs opacity-80 leading-snug">{alert.message}</p>
              </div>
            </div>
          ))}
        </div>
        {/* Alerts title here */}
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-primary" />
            <h3 className="text-sm font-bold uppercase tracking-wider text-primary flex items-center gap-2">
              Alerts
            </h3>
          </div>
          </>
      )}
    </div>
  );
}
