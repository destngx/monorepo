"use client";

import { Card, CardContent, CardHeader, CardTitle } from "./card";
import { TrendingUp, AlertCircle, ShieldCheck, HelpCircle, Users, Coins } from "lucide-react";
import { MaskedBalance } from "../ui/masked-balance";

interface PatternData {
  forecast?: { message: string; amount: number; status: 'on-track' | 'at-risk' };
  anomalies?: Array<{ date: string; title: string; amount: number; reason: string }>;
  portfolioScore?: { score: number; breakdown: string };
  whatIf?: { scenario: string; impact: string };
  benchmarking?: { message: string; percentile: number };
  investments?: { title: string; tactic: string; opportunity: string };
}

export function AIInsightPatterns({ patterns }: { patterns: PatternData | undefined }) {
  if (!patterns) return null;

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 margin-bottom-4">
      {/* Spending Forecast */}
      {patterns.forecast && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-primary ">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <TrendingUp className="h-4 w-4 text-primary" />
              Spending Forecast
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 font-medium leading-relaxed">
              {patterns.forecast.message}
            </p>
            <div className={`mt-3 text-xs font-bold uppercase tracking-wider ${
              patterns.forecast.status === 'at-risk' ? 'text-rose-500' : 'text-emerald-500'
            }`}>
              Status: {patterns.forecast.status.replace('-', ' ')}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Portfolio Health Score */}
      {patterns.portfolioScore && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-indigo-500">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <ShieldCheck className="h-4 w-4 text-indigo-500" />
              Portfolio Health
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center gap-4">
              <div className="text-3xl font-black text-indigo-500">{patterns.portfolioScore.score}</div>
              <p className="text-xs text-zinc-500 font-medium leading-snug">
                {patterns.portfolioScore.breakdown}
              </p>
            </div>
            <div className="w-full bg-zinc-100 dark:bg-zinc-900 h-1.5 rounded-full overflow-hidden">
              <div 
                className="bg-indigo-500 h-full rounded-full transition-all duration-1000" 
                style={{ width: `${patterns.portfolioScore.score}%` }} 
              />
            </div>
          </CardContent>
        </Card>
      )}

      {/* What-If Scenarios */}
      {patterns.whatIf && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-emerald-500">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <HelpCircle className="h-4 w-4 text-emerald-500" />
              "What-If" Analysis
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm font-bold text-zinc-800 dark:text-zinc-200 italic mb-2">"{patterns.whatIf.scenario}"</p>
            <p className="text-xs text-emerald-600 dark:text-emerald-400 font-bold bg-emerald-50 dark:bg-emerald-950/20 p-2 rounded-lg">
              Potential Impact: {patterns.whatIf.impact}
            </p>
          </CardContent>
        </Card>
      )}

      {/* Anomaly Detection */}
      {patterns.anomalies && patterns.anomalies.length > 0 && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-amber-500">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-amber-500" />
              Anomaly Detected
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {patterns.anomalies.map((a, i) => (
              <div key={i} className="space-y-1">
                <div className="flex justify-between items-center text-xs">
                  <span className="font-bold text-zinc-800 dark:text-zinc-200">{a.title}</span>
                  <span className="text-amber-600 tabular-nums"><MaskedBalance amount={a.amount} /></span>
                </div>
                <p className="text-[10px] text-zinc-500">{a.reason}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Peer Benchmarking */}
      {patterns.benchmarking && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-blue-500">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <Users className="h-4 w-4 text-blue-500" />
              Community Benchmarking
            </CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-zinc-600 dark:text-zinc-400 font-medium mb-3">
              {patterns.benchmarking.message}
            </p>
            <div className="flex items-center gap-2">
              <div className="text-xl font-black text-blue-500">Top {100 - patterns.benchmarking.percentile}%</div>
              <span className="text-[10px] text-zinc-500 uppercase font-bold tracking-tighter">Savings percentile</span>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Investment Insights */}
      {patterns.investments && (
        <Card className="border-none bg-white dark:bg-zinc-950 shadow-sm border-l-4 border-l-violet-500">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-bold flex items-center gap-2">
              <Coins className="h-4 w-4 text-violet-500" />
              Investment Strategy
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <h4 className="text-xs font-bold text-violet-600 dark:text-violet-400 uppercase tracking-wide">
              {patterns.investments.title}
            </h4>
            <p className="text-xs text-zinc-600 dark:text-zinc-400 font-medium leading-relaxed">
              {patterns.investments.tactic}
            </p>
            <div className="p-2 bg-violet-50 dark:bg-violet-950/20 rounded-lg border border-violet-100 dark:border-violet-900/50">
              <p className="text-[10px] uppercase font-black text-violet-700 dark:text-violet-300 mb-1">Tactical Opportunity</p>
              <p className="text-xs font-bold text-zinc-800 dark:text-zinc-200">{patterns.investments.opportunity}</p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
