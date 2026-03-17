'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Sparkles, TrendingDown, Calendar, Wallet, Target, Info, AlertTriangle, CheckCircle2 } from 'lucide-react';
import { formatVND } from '@wealth-management/utils';
import { Button } from '@/components/ui/button';
import { XAxis, YAxis, Tooltip, ResponsiveContainer, AreaChart, Area } from 'recharts';
import { Badge } from '@/components/ui/badge';

interface AdvisorData {
  briefing: string;
  suggestions: Array<{ category: string; suggestedLimit: number; rationale: string; currentAverage: number }>;
  forecast: {
    projection: Array<{ date: string; balance: number }>;
    annotations: Array<{ date: string; message: string }>;
  };
  alerts: Array<{
    type: 'overspend' | 'spike' | 'bill' | 'opportunity' | 'goal';
    title: string;
    message: string;
    urgency: 'critical' | 'warning' | 'info';
  }>;
  goalImpact: string;
}

export function AIBudgetAdvisorView({ data }: { data?: AdvisorData }) {
  if (!data)
    return (
      <div className="flex flex-col items-center justify-center py-24 text-center space-y-4">
        <div className="relative">
          <Sparkles className="h-12 w-12 text-primary/20 animate-pulse" />
          <div className="absolute inset-0 bg-primary/5 blur-3xl rounded-full" />
        </div>
        <p className="text-sm font-medium text-muted-foreground italic">
          Your AI Advisor is synchronizing with your financial history...
        </p>
      </div>
    );

  const getUrgencyStyles = (urgency: string) => {
    switch (urgency) {
      case 'critical':
        return 'bg-rose-50 border-rose-100 text-rose-900 dark:bg-rose-950/20 dark:border-rose-900/50 dark:text-rose-200';
      case 'warning':
        return 'bg-amber-50 border-amber-100 text-amber-900 dark:bg-amber-950/20 dark:border-amber-900/50 dark:text-amber-200';
      default:
        return 'bg-blue-50 border-blue-100 text-blue-900 dark:bg-blue-950/20 dark:border-blue-900/50 dark:text-blue-200';
    }
  };

  const getAlertIcon = (type: string) => {
    switch (type) {
      case 'overspend':
        return <AlertTriangle className="h-4 w-4" />;
      case 'spike':
        return <TrendingDown className="h-4 w-4" />;
      case 'bill':
        return <Calendar className="h-4 w-4" />;
      case 'goal':
        return <Target className="h-4 w-4" />;
      default:
        return <Info className="h-4 w-4" />;
    }
  };

  return (
    <div className="space-y-8 pb-12">
      {/* 30-Day Forward Projection */}
      <Card className="border-none shadow-xl shadow-zinc-200/50 dark:shadow-none overflow-hidden">
        <CardHeader className="bg-zinc-950 text-white p-6">
          <div className="flex justify-between items-start">
            <div className="space-y-1">
              <CardTitle className="text-lg font-bold flex items-center gap-2">
                <TrendingDown className="h-5 w-5 text-primary" />
                Predictive Cash Flow
              </CardTitle>
              <CardDescription className="text-zinc-400 font-medium">Next 30-day balance forecast</CardDescription>
            </div>
            <Badge variant="outline" className="border-zinc-800 text-zinc-400 bg-zinc-900/50 backdrop-blur-sm">
              AI Projected
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="bg-zinc-950 p-0">
          <div className="h-[250px] w-full mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data.forecast.projection}>
                <defs>
                  <linearGradient id="colorBalance" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#6366f1" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#6366f1" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <XAxis dataKey="date" hide />
                <YAxis hide domain={['auto', 'auto']} />
                <Tooltip
                  contentStyle={{ backgroundColor: '#09090b', border: '1px solid #27272a', borderRadius: '12px' }}
                  itemStyle={{ color: '#fff' }}
                  labelStyle={{ color: '#71717a' }}
                  formatter={(v: any) => [formatVND(v || 0), 'Projected Balance']}
                />
                <Area
                  type="monotone"
                  dataKey="balance"
                  stroke="#6366f1"
                  strokeWidth={3}
                  fillOpacity={1}
                  fill="url(#colorBalance)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          <div className="p-6 pt-0 space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {data.forecast.annotations.map((ann, i) => (
                <div key={i} className="flex gap-3 items-start p-3 bg-zinc-900/50 border border-zinc-800 rounded-xl">
                  <div className="p-1.5 bg-zinc-800 rounded-lg shrink-0">
                    <Info className="h-3 w-3 text-primary" />
                  </div>
                  <div className="space-y-0.5">
                    <p className="text-[10px] font-black text-zinc-500 uppercase tracking-tighter">{ann.date}</p>
                    <p className="text-xs text-zinc-300 font-medium leading-relaxed">{ann.message}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* AI Budget Builder */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div className="space-y-1">
            <h3 className="text-lg font-bold flex items-center gap-2">
              <Wallet className="h-5 w-5 text-indigo-500" />
              AI Budget Builder
            </h3>
            <p className="text-muted-foreground text-xs font-medium">
              Suggestions based on your last 6 months of habits
            </p>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="h-8 text-xs font-bold rounded-lg px-4 bg-primary text-primary-foreground border-none hover:bg-primary/90"
          >
            Apply All Changes
          </Button>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {data.suggestions.map((s, i) => (
            <Card
              key={i}
              className="border-none shadow-sm hover:ring-2 hover:ring-primary/10 transition-all border-l-4 border-l-indigo-500"
            >
              <CardContent className="p-6 flex gap-6">
                <div className="flex-1 space-y-4">
                  <div className="flex justify-between items-start">
                    <div>
                      <h4 className="text-sm font-black text-zinc-800 dark:text-zinc-200">{s.category}</h4>
                      <p className="text-[10px] text-muted-foreground font-bold italic">
                        Avg spend: {formatVND(s.currentAverage)}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="text-[10px] uppercase font-black text-muted-foreground tracking-tighter">
                        Proposed Limit
                      </p>
                      <p className="text-sm font-black text-indigo-600 tabular-nums">{formatVND(s.suggestedLimit)}</p>
                    </div>
                  </div>
                  <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed font-medium bg-zinc-50 dark:bg-zinc-900/50 p-3 rounded-lg border border-zinc-100 dark:border-zinc-800">
                    {s.rationale}
                  </p>
                </div>
                <div className="flex flex-col gap-2 shrink-0">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 rounded-lg bg-emerald-50 dark:bg-emerald-950/20 text-emerald-600 dark:text-emerald-400 border border-emerald-100 dark:border-emerald-900/50"
                  >
                    <CheckCircle2 className="h-4 w-4" />
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </div>

      {/* Adaptive Alerts & Goals */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-4">
          <h3 className="text-[10px] font-black uppercase text-muted-foreground tracking-widest px-1">
            Urgent Alerts & Nudges
          </h3>
          <div className="space-y-3">
            {data.alerts.map((alert, i) => (
              <div
                key={i}
                className={`p-4 rounded-xl border flex gap-4 items-start transition-all hover:translate-x-1 ${getUrgencyStyles(alert.urgency)}`}
              >
                <div className="shrink-0 pt-0.5">{getAlertIcon(alert.type)}</div>
                <div className="space-y-1">
                  <h4 className="text-xs font-black uppercase tracking-wide">{alert.title}</h4>
                  <p className="text-xs opacity-90 leading-relaxed font-medium">{alert.message}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="space-y-4">
          <h3 className="text-[10px] font-black uppercase text-muted-foreground tracking-widest px-1">
            Goal Integration
          </h3>
          <Card className="border-none bg-indigo-50/50 dark:bg-indigo-950/20 shadow-sm border-t-4 border-t-indigo-500 overflow-hidden">
            <CardContent className="p-6 relative">
              <div className="space-y-4 z-10 relative">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-indigo-500/10 rounded-lg">
                    <Target className="h-4 w-4 text-indigo-600" />
                  </div>
                  <h4 className="text-sm font-bold text-indigo-900 dark:text-indigo-100">Knock-on Effects</h4>
                </div>
                <p className="text-xs text-indigo-800/80 dark:text-indigo-200/80 font-medium leading-relaxed italic">
                  {data.goalImpact}
                </p>
              </div>
              <div className="absolute bottom-0 right-0 p-4 opacity-5">
                <TrendingDown className="h-16 w-16" />
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
