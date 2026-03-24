'use client';

import { useState } from 'react';
import useSWR from 'swr';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip as ReTooltip, 
  ResponsiveContainer, 
  Cell,
  ReferenceLine 
} from 'recharts';
import { 
  Layers, 
  Info, 
  RefreshCcw, 
  Search, 
  Globe, 
  Sparkles,
  Zap,
  Calendar,
  BarChart3
} from 'lucide-react';
import { 
  GLASS_CARD, 
  SeasonalityStatsTable 
} from './market-pulse-dashboard';
import { AIDataInsight } from '@/components/dashboard/ai-data-insight';
import { Badge } from '@/components/ui/badge';

const fetcher = (url: string) => fetch(url).then((res) => res.json());

export function SeasonalPatternsDashboard() {
  const [market, setMarket] = useState<'US' | 'VN'>('VN');
  const [timeframe] = useState('1d'); // Seasonality is most accurate on Daily data
  const { data, isLoading, mutate } = useSWR(`/api/market-pulse?timeframe=${timeframe}`, fetcher);

  const marketData = market === 'US' ? data?.us : data?.vn;
  const seasonality = marketData?.technicals?.seasonality || [];

  // Pre-calculate best/worst for safer rendering
  const validSeasonality = seasonality.filter((s: any) => s.return !== undefined && s.score !== undefined);
  const bestDay = validSeasonality.length > 0 ? [...validSeasonality].sort((a: any, b: any) => b.score - a.score)[0] : null;
  const worstDay = validSeasonality.length > 0 ? [...validSeasonality].sort((a: any, b: any) => a.score - b.score)[0] : null;

  const handleRefresh = async () => {
    await mutate();
  };

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[400px] gap-4">
        <RefreshCcw className="w-8 h-8 animate-spin text-indigo-500" />
        <p className="text-sm font-mono text-zinc-500 animate-pulse tracking-widest uppercase">
          Calculating Seasonal Probabilities...
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-8 animate-in fade-in duration-700">
      {/* Header Info */}
      <div className={`p-6 rounded-3xl ${GLASS_CARD} flex flex-col md:flex-row justify-between items-start md:items-center gap-6`}>
        <div className="space-y-1">
          <h2 className="text-2xl font-black tracking-tighter uppercase flex items-center gap-3">
            <Layers className="w-6 h-6 text-emerald-500" />
            Seasonal Alamanac
          </h2>
          <p className="text-[10px] uppercase font-bold text-zinc-500 tracking-widest">
            Historical probability engine based on 1 year of session data
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-4">
          <div className="flex bg-zinc-100 dark:bg-zinc-800/50 p-1 rounded-xl">
            {(['US', 'VN'] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMarket(m)}
                className={`px-4 py-1.5 text-xs font-bold rounded-lg transition-all ${
                  market === m ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900 shadow-lg' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'
                }`}
              >
                {m} ALPHA
              </button>
            ))}
          </div>
          <button 
             onClick={handleRefresh}
             className="p-2 rounded-xl bg-zinc-100 dark:bg-zinc-800/50 text-zinc-500 hover:text-indigo-500 transition-colors"
          >
            <RefreshCcw className="w-4 h-4" />
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
        <div className="xl:col-span-2 space-y-8">
          {/* Main Seasonal Chart */}
          <Card className={`${GLASS_CARD} p-6 h-[400px]`}>
            <div className="flex items-center justify-between mb-6">
              <div className="flex items-center gap-2">
                <BarChart3 className="w-4 h-4 text-indigo-500" />
                <span className="text-xs font-black uppercase tracking-widest text-zinc-400">Day of Week Performance</span>
              </div>
              <Badge variant="outline" className="text-[10px] font-mono border-zinc-800">1Y LOOKBACK</Badge>
            </div>
            
            <div className="h-[300px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={seasonality.filter((s: any) => s.type === 'day')}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#27272a" />
                  <XAxis 
                    dataKey="name" 
                    axisLine={false} 
                    tickLine={false} 
                    tick={{ fill: '#71717a', fontSize: 10, fontWeight: 700 }} 
                  />
                  <YAxis 
                    axisLine={false} 
                    tickLine={false} 
                    tick={{ fill: '#71717a', fontSize: 10, fontWeight: 700 }}
                    tickFormatter={(v) => `${v}%`}
                  />
                  <ReTooltip
                    contentStyle={{ backgroundColor: '#09090b', borderColor: '#27272a', borderRadius: '12px', fontSize: '10px' }}
                    itemStyle={{ color: '#fff', fontWeight: 800 }}
                    cursor={{ fill: 'rgba(255,255,255,0.03)' }}
                  />
                  <ReferenceLine y={0} stroke="#3f3f46" />
                  <Bar dataKey="return" radius={[4, 4, 0, 0]}>
                    {seasonality.filter((s: any) => s.type === 'day').map((entry: any, index: number) => (
                      <Cell 
                        key={`cell-${index}`} 
                        fill={(entry.return ?? 0) >= 0 ? (market === 'US' ? '#6366f1' : '#10b981') : '#f43f5e'} 
                        fillOpacity={0.8}
                      />
                    ))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            </div>
          </Card>

          {/* Detailed Stats Table */}
          <SeasonalityStatsTable seasonality={seasonality} />
        </div>

        <div className="space-y-8">
           {/* AI Seasonality Report */}
           <Card className={`${GLASS_CARD} border-emerald-500/20 shadow-emerald-500/5`}>
             <CardHeader className="border-b border-zinc-800/50 pb-4">
                <div className="flex items-center justify-between">
                   <CardTitle className="text-sm font-black uppercase tracking-widest flex items-center gap-2">
                     <Sparkles className="w-4 h-4 text-emerald-400" />
                     Smart Money Report
                   </CardTitle>
                   <AIDataInsight 
                      type="Seasonal Intelligence"
                      description={`Comprehensive seasonal pattern analysis for the ${market} market based on historical probability distributions.`}
                      data={seasonality}
                      market={market}
                    />
                </div>
                <CardDescription className="text-[10px] uppercase font-mono text-zinc-500 mt-1">
                   AI-Driven Probability Synthesis
                </CardDescription>
             </CardHeader>
             <CardContent className="pt-6">
                <div className="space-y-6">
                   <div className="space-y-2">
                      <div className="text-[10px] font-bold text-indigo-400 uppercase flex items-center gap-1.5">
                         <Zap className="w-3 h-3" /> Highest Edge Found
                      </div>
                      {bestDay ? (
                        <div className="p-4 rounded-2xl bg-indigo-500/10 border border-indigo-500/20">
                           <div className="flex justify-between items-center mb-1">
                              <span className="text-lg font-black">{bestDay.name}</span>
                              <Badge className="bg-indigo-500 text-white border-none text-[10px]">{bestDay.winRate || 0}% WR</Badge>
                           </div>
                           <p className="text-xs text-zinc-400 leading-relaxed">
                              Historical data shows a significant bullish bias on this day, with an average return of <span className="text-emerald-400 font-bold">+{bestDay.return?.toFixed(2) || '0.00'}%</span>.
                           </p>
                        </div>
                      ) : (
                        <div className="text-xs text-zinc-500 italic p-4 rounded-2xl bg-zinc-500/5 border border-zinc-500/10">
                           Insufficient data for pattern detection...
                        </div>
                      )}
                   </div>

                   <div className="space-y-2">
                      <div className="text-[10px] font-bold text-rose-400 uppercase flex items-center gap-1.5">
                         <Search className="w-3 h-3" /> Risk Avoidance
                      </div>
                      {worstDay ? (
                        <div className="p-4 rounded-2xl bg-rose-500/10 border border-rose-500/20">
                           <div className="flex justify-between items-center mb-1">
                              <span className="text-lg font-black">{worstDay.name}</span>
                              <Badge className="bg-rose-500 text-white border-none text-[10px]">{worstDay.winRate || 0}% WR</Badge>
                           </div>
                           <p className="text-xs text-zinc-400 leading-relaxed">
                              High distribution risk noted. Historical returns average <span className="text-rose-400 font-bold">{worstDay.return?.toFixed(2) || '0.00'}%</span> with elevated volatility.
                           </p>
                        </div>
                      ) : (
                        <div className="text-xs text-zinc-500 italic p-4 rounded-2xl bg-zinc-500/5 border border-zinc-500/10">
                           No high-risk patterns identified...
                        </div>
                      )}
                   </div>

                   <div className="pt-4 border-t border-zinc-800/50 flex items-center justify-between text-[10px] font-bold text-zinc-500">
                      <div className="flex items-center gap-1.5">
                        <Globe className="w-3 h-3" /> {market} REGIME
                      </div>
                      <div className="flex items-center gap-1.5 uppercase">
                        <Calendar className="w-3 h-3" /> {validSeasonality.length || 252} Sessions Analyzed
                      </div>
                   </div>
                </div>
             </CardContent>
           </Card>
        </div>
      </div>
    </div>
  );
}
