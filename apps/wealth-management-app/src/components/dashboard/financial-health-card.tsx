"use client";

import { useState, useEffect, useMemo, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Account } from "@wealth-management/types";
import { Transaction } from "@wealth-management/types";
import { Loan } from "@wealth-management/types";
import { HeartPulse, CheckCircle2, AlertCircle, TrendingUp, Sparkles } from "lucide-react";
import { formatVND } from "@wealth-management/utils";

interface Props {
  accounts: Account[];
  transactions: Transaction[];
  loans: Loan[];
}

interface HealthSignal {
  type: "good" | "warn" | "bad";
  title: string;
  detail: string;
}

function getRecentMonths(transactions: Transaction[], n: number) {
  const now = new Date();
  const results: { month: number; year: number; income: number; expense: number }[] = [];
  for (let i = 0; i < n; i++) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const m = d.getMonth();
    const y = d.getFullYear();
    const txns = transactions.filter(t => {
      const td = new Date(t.date);
      return td.getMonth() === m && td.getFullYear() === y;
    });
    results.push({
      month: m, year: y,
      income: txns.reduce((s, t) => s + (t.deposit || 0), 0),
      expense: txns.reduce((s, t) => s + (t.payment || 0), 0),
    });
  }
  return results;
}

export function FinancialHealthCard({ accounts, transactions, loans }: Props) {
  const [score, setScore] = useState<number | null>(null);
  const [signals, setSignals] = useState<HealthSignal[]>([]);
  const [loading, setLoading] = useState(true);
  const lastFetchRef = useRef<string>("");

  const months = useMemo(() => getRecentMonths(transactions, 3), [transactions]);

  useEffect(() => {
    const payloadSignature = JSON.stringify({
      acctCount: accounts.length,
      txnVol: months.reduce((acc, m) => acc + m.income + m.expense, 0),
      loanCount: loans.length
    });

    if (lastFetchRef.current === payloadSignature) return;

    const fetchHealth = async () => {
      lastFetchRef.current = payloadSignature;
      setLoading(true);
      try {
        const res = await fetch('/api/ai/financial-health', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ accounts, months, loans })
        });
        const data = await res.json();
        if (data.score !== undefined) {
          setScore(data.score);
          setSignals(data.signals || []);
        }
      } catch (err) {
        console.error("AI Health Fetch Error:", err);
      } finally {
        setLoading(false);
      }
    };

    fetchHealth();
  }, [accounts, months, loans]);

  const scoreColor = score === null ? "text-muted-foreground" : score >= 75 ? "text-emerald-500" : score >= 50 ? "text-amber-500" : "text-red-500";
  const goods = signals.filter(s => s.type === "good");
  const warnings = signals.filter(s => s.type !== "good");

  return (
    <Card className="shadow-sm h-full">
      <CardHeader className="pb-3 border-b">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-primary/10 rounded-lg">
              <HeartPulse className="h-4 w-4 text-primary" />
            </div>
            <div>
              <CardTitle className="text-base font-bold">Lộc Phát Tài Health Review</CardTitle>
              <p className="text-[10px] text-muted-foreground font-medium uppercase tracking-tight">Financial Health Score</p>
            </div>
          </div>
          <div className={`text-2xl font-bold tabular-nums flex items-center gap-2 ${scoreColor}`}>
            {loading ? (
              <div className="flex items-center gap-2 animate-pulse">
                <Sparkles className="h-4 w-4 text-indigo-400" />
                <span className="text-xs font-medium">Thinking...</span>
              </div>
            ) : (
              <>
                {score ?? "--"}
                <span className="text-sm font-normal text-muted-foreground">/100</span>
              </>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="pt-4 space-y-3">
        {loading ? (
          <div className="space-y-4">
            {[1, 2, 3].map(i => (
              <div key={i} className="flex items-start gap-2.5 animate-pulse">
                <div className="h-4 w-4 rounded-full bg-muted" />
                <div className="space-y-2 flex-1">
                  <div className="h-3 w-1/3 bg-muted rounded" />
                  <div className="h-2 w-full bg-muted rounded" />
                </div>
              </div>
            ))}
          </div>
        ) : (
          <>
            {goods.length === 0 && warnings.length === 0 && (
              <p className="text-xs text-muted-foreground italic">No analysis data available yet.</p>
            )}
            
            {goods.map((s, i) => (
              <div key={i} className="flex items-start gap-2.5">
                <CheckCircle2 className="h-4 w-4 text-emerald-500 mt-0.5 shrink-0" />
                <div>
                  <p className="text-sm font-semibold leading-none mb-1">{s.title}</p>
                  <p className="text-xs text-muted-foreground">{s.detail}</p>
                </div>
              </div>
            ))}
            {warnings.map((s, i) => (
              <div key={i} className="flex items-start gap-2.5">
                <AlertCircle className={`h-4 w-4 mt-0.5 shrink-0 ${s.type === "bad" ? "text-red-500" : "text-amber-500"}`} />
                <div>
                  <p className="text-sm font-semibold leading-none mb-1">{s.title}</p>
                  <p className="text-xs text-muted-foreground">{s.detail}</p>
                </div>
              </div>
            ))}
          </>
        )}
        
        <div className="pt-2 border-t mt-2 flex items-center justify-between gap-1.5 text-[10px] text-muted-foreground">
          <div className="flex items-center gap-1.5 ">
            <TrendingUp className="h-3 w-3" />
            Analysis based on last 3 months
          </div>
          <div className="flex items-center gap-1 text-indigo-500/70 font-medium">
            <Sparkles className="h-2.5 w-2.5" />
            AI Powered
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
