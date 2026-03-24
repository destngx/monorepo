'use client';

import { useState } from 'react';
import useSWR from 'swr';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  History,
  ArrowRight,
  BarChart3,
  Compass,
  Zap,
  Target,
  Activity,
  Clock,
  ArrowUpRight,
  ArrowDownRight,
  Info
} from 'lucide-react';
import {
  GLASS_CARD,
  TERMINAL_FONT,
  MarketSignalCard,
  IntelligenceBanner,
  TimeframeRelationshipGrid,
  TechnicalAnalysisView
} from './market-pulse-dashboard';
import { Label } from '@/components/ui/label';

const fetcher = (url: string) => fetch(url).then((res) => res.json());

const TIMEFRAMES = [
  { value: '1h', label: '1H', desc: 'Scalp / Intraday' },
  { value: '4h', label: '4H', desc: 'Local Trend' },
  { value: '1d', label: '1D', desc: 'Daily Session' },
  { value: '1w', label: '1W', desc: 'Structural Trend' },
];

export function MultiTimeframeDashboard() {
  const [selectedTFs, setSelectedTFs] = useState<string[]>(['1h', '4h', '1d']);
  const [market, setMarket] = useState<'US' | 'VN'>('US');

  const { data: d1h, isLoading: loading1h } = useSWR(selectedTFs.includes('1h') ? `/api/market-pulse?timeframe=1h` : null, fetcher);
  const { data: d4h, isLoading: loading4h } = useSWR(selectedTFs.includes('4h') ? `/api/market-pulse?timeframe=4h` : null, fetcher);
  const { data: d1d, isLoading: loading1d } = useSWR(selectedTFs.includes('1d') ? `/api/market-pulse?timeframe=1d` : null, fetcher);
  const { data: d1w, isLoading: loading1w } = useSWR(selectedTFs.includes('1w') ? `/api/market-pulse?timeframe=1w` : null, fetcher);

  const tfData: Record<string, { data: any; isLoading: boolean }> = {
    '1h': { data: d1h, isLoading: loading1h },
    '4h': { data: d4h, isLoading: loading4h },
    '1d': { data: d1d, isLoading: loading1d },
    '1w': { data: d1w, isLoading: loading1w },
  };

  const toggleTF = (tf: string) => {
    setSelectedTFs((prev) =>
      prev.includes(tf) ? prev.filter((t) => t !== tf) : [...prev, tf]
    );
  };

  const getMarketData = (tf: string) => (market === 'US' ? tfData[tf]?.data?.us : tfData[tf]?.data?.vn);

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      {/* Header Controls */}
      <div className={`p-6 rounded-3xl ${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 flex flex-col md:flex-row justify-between items-start md:items-center gap-6`}>
        <div className="space-y-1">
          <h2 className="text-2xl font-black tracking-tighter uppercase flex items-center gap-3">
            <History className="w-6 h-6 text-indigo-500" />
            Confluence Matrix
          </h2>
          <p className="text-[10px] uppercase font-bold text-zinc-500 tracking-widest">
            Cross-timeframe confluence analysis for Institutional Grade Execution
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-6">
          <div className="flex bg-zinc-100 dark:bg-zinc-800/50 p-1 rounded-xl">
            {(['US', 'VN'] as const).map((m) => (
              <button
                key={m}
                onClick={() => setMarket(m)}
                className={`px-4 py-1.5 text-xs font-bold rounded-lg transition-all ${market === m ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-900 shadow-lg' : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200'
                  }`}
              >
                {m} CORE
              </button>
            ))}
          </div>

          <div className="flex items-center gap-4">
            {TIMEFRAMES.map((tf) => (
              <div key={tf.value} className="flex items-center space-x-2 bg-zinc-50 dark:bg-zinc-900/40 px-3 py-1.5 rounded-lg border border-zinc-200 dark:border-zinc-800/50">
                <input
                  type="checkbox"
                  id={`tf-${tf.value}`}
                  checked={selectedTFs.includes(tf.value)}
                  onChange={() => toggleTF(tf.value)}
                  className="w-3.5 h-3.5 rounded border-zinc-400 text-indigo-600 focus:ring-indigo-500 bg-transparent"
                />
                <Label
                  htmlFor={`tf-${tf.value}`}
                  className="text-[10px] font-black cursor-pointer uppercase tracking-tight"
                >
                  {tf.label}
                </Label>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Confluence Summary Table */}
      <Card className={`${GLASS_CARD} overflow-hidden`}>
        <div className="overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-zinc-900/40 border-b border-zinc-800/50 text-[10px] font-bold uppercase tracking-widest text-zinc-500">
                <th className="py-4 px-6">Interval</th>
                <th className="py-4 px-6">Market Cycle</th>
                <th className="py-4 px-6">Indicators</th>
                <th className="py-4 px-6">Current Signal</th>
                <th className="py-4 px-6">R:R</th>
                <th className="py-4 px-6">Confidence</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800/20">
              {selectedTFs.sort((a, b) => TIMEFRAMES.findIndex(x => x.value === a) - TIMEFRAMES.findIndex(x => x.value === b)).map((tf) => {
                const data = getMarketData(tf);
                const isLoading = tfData[tf].isLoading;

                if (isLoading) {
                  return (
                    <tr key={tf} className="animate-pulse">
                      <td colSpan={6} className="py-4 px-6 text-zinc-500 italic text-[10px] font-mono">
                        FETCHING {tf} DATA...
                      </td>
                    </tr>
                  );
                }

                if (!data) return null;

                const signals = data.technicals?.signals;
                const signalAction = signals?.action || 'AVOID';
                const isMarkup = data.technicals?.cycle?.phase === 'Markup';
                const isMarkdown = data.technicals?.cycle?.phase === 'Mark-Down' || data.technicals?.cycle?.phase === 'Decline';
                const signalColor = signalAction === 'LONG' ? 'text-emerald-500' : signalAction === 'SHORT' ? 'text-rose-500' : 'text-zinc-500';
                const signalBg = signalAction === 'LONG' ? 'bg-emerald-500' : signalAction === 'SHORT' ? 'bg-rose-500' : 'bg-zinc-500';

                return (
                  <tr key={tf} className="hover:bg-zinc-50 dark:hover:bg-white/[0.02] transition-colors group">
                    <td className="py-6 px-6">
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-xl bg-zinc-900 flex items-center justify-center border border-zinc-800 text-xs font-black text-indigo-400 group-hover:scale-110 transition-transform">
                          {tf.toUpperCase()}
                        </div>
                        <div className="hidden sm:block">
                          <div className="text-[10px] font-bold text-zinc-400 uppercase tracking-tighter">Horizon</div>
                          <div className="text-xs font-mono font-bold">{TIMEFRAMES.find(t => t.value === tf)?.desc}</div>
                        </div>
                      </div>
                    </td>
                    <td className="py-6 px-6">
                      <div className="space-y-1">
                        <Badge className={`${isMarkup ? 'bg-emerald-500 text-white' : isMarkdown ? 'bg-rose-500 text-white' : 'bg-blue-500 text-white'} border-none text-[8px] px-1 h-4`}>
                          {(data.technicals?.cycle?.phase || 'N/A').toUpperCase()}
                        </Badge>
                        <div className="text-[10px] text-zinc-500 font-medium leading-tight max-w-[150px]">
                          {data.technicals?.cycle?.description || 'Awaiting Analysis'}
                        </div>
                      </div>
                    </td>
                    <td className="py-6 px-6">
                      <div className="flex flex-col gap-1">
                        <div className="flex items-center gap-2">
                          <span className="text-[10px] font-mono text-zinc-500">RSI:</span>
                          <span className={`text-[10px] font-black ${(data.technicals?.indicators?.rsi ?? 50) > 70 ? 'text-rose-500' : (data.technicals?.indicators?.rsi ?? 50) < 30 ? 'text-emerald-500' : 'text-zinc-400'}`}>
                            {data.technicals?.indicators?.rsi?.toFixed(1) || 'N/A'}
                          </span>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="text-[10px] font-mono text-zinc-500">EMA50:</span>
                          <span className="text-[10px] font-black text-zinc-400">{data.technicals?.indicators?.ema50?.toLocaleString() || 'N/A'}</span>
                        </div>
                      </div>
                    </td>
                    <td className="py-6 px-6">
                      <div className="flex items-center gap-3">
                        {signalAction === 'LONG' ? <ArrowUpRight className="w-4 h-4 text-emerald-500" /> : signalAction === 'SHORT' ? <ArrowDownRight className="w-4 h-4 text-rose-500" /> : <Clock className="w-4 h-4 text-zinc-500" />}
                        <div className="flex flex-col">
                          <span className={`text-xs font-black ${signalColor}`}>{signalAction}</span>
                          <span className="text-[9px] text-zinc-500 font-mono italic">{signals?.actionVi || '-'}</span>
                        </div>
                      </div>
                    </td>
                    <td className="py-6 px-6 font-mono font-bold text-amber-500 text-xs text-center sm:text-left">
                      1:{signals?.rr || '-'}
                    </td>
                    <td className="py-6 px-6">
                      <div className="space-y-1.5">
                        <div className="flex justify-between text-[9px] font-black text-zinc-500">
                          <span>SCORE</span>
                          <span>{signals?.confidence || 0}%</span>
                        </div>
                        <div className="w-20 h-1 bg-zinc-800 rounded-full overflow-hidden">
                          <div
                            className={`h-full ${signalBg}`}
                            style={{ width: `${signals?.confidence || 0}%` }}
                          />
                        </div>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Comparison View Grid */}
      <div className="grid grid-cols-1 gap-8">
        {selectedTFs.map((tf) => {
          const data = getMarketData(tf);
          if (!data) return null;

          return (
            <div key={tf} className="space-y-6">
              <div className="flex items-center gap-4 px-2">
                <Badge variant="outline" className="bg-indigo-500/10 text-indigo-500 border-indigo-500/20 font-mono px-3 py-1">
                  TIMEFRAME: {tf.toUpperCase()}
                </Badge>
                <div className="h-px flex-1 bg-zinc-800" />
              </div>

              <div className="grid grid-cols-1 gap-6">
                <MarketSignalCard signals={data.technicals.signals} market={market} />
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  <div className="space-y-6">
                    <TechnicalAnalysisView technicals={data.technicals} market={market} showSeasonality={false} />
                  </div>
                  <div className="space-y-6">
                    <TimeframeRelationshipGrid
                      relationships={data.technicals.timeframeRelationships}
                      entryScore={data.technicals.entryTimingScore}
                    />
                    <IntelligenceBanner
                      scenarios={data.scenarios}
                      capitalFlow={data.capitalFlow}
                      market={market}
                      timeframe={tf}
                    />
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
