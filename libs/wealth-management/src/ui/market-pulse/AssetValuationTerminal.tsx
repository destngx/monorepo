'use client';

import { Layers, Binary, BarChart3, PieChart, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { Card, CardHeader, CardTitle, CardContent } from '../card';
import { Badge } from '../badge';
import { AIDataInsight } from '../ai-data-insight';
import { GLASS_CARD } from './constants';

export function AssetValuationTerminal({ valuation, market }: { valuation?: any; market: string }) {
  if (!valuation) return null;

  const accentColor = market === 'US' ? 'indigo' : 'emerald';

  return (
    <div className="pt-8 border-t border-zinc-100 dark:border-zinc-800/50 mt-10">
      <div className="flex items-center justify-between gap-2 mb-6">
        <div className="flex items-center gap-2">
          <Layers className={`w-5 h-5 ${market === 'US' ? 'text-indigo-500' : 'text-emerald-500'}`} />
          <h3 className="text-xl font-bold tracking-tight uppercase">Valuation Decision Terminal: {market}</h3>
        </div>
        <AIDataInsight
          type="Valuation Terminal"
          description={`Financial valuation models for the ${market} market, featuring DCF projections, Monte Carlo probability distributions, and competitive sector benchmarking.`}
          data={valuation}
          market={market}
          timeframe="1D" // Valuation models span larger horizons
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* DCF Valuation */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50`}>
          <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
            <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
              <Binary className="w-3.5 h-3.5" /> DCF Model Scenarios
            </CardTitle>
          </CardHeader>
          <CardContent className="p-5 space-y-4">
            {(valuation.dcf || []).map((item: any, idx: number) => (
              <div
                key={idx}
                className="space-y-3 pb-4 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0"
              >
                <div className="flex items-center justify-between">
                  <span className="font-bold text-sm tracking-tight">{item.symbol}</span>
                  <Badge
                    className={`${item.upside > 0 ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'} border-none uppercase text-[9px]`}
                  >
                    {item.upside > 0 ? '+' : ''}
                    {item.upside.toFixed(1)}% Potential
                  </Badge>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="text-xl font-mono font-bold tracking-tighter">
                    {market === 'VN' && item.symbol !== 'USD/VND'
                      ? (item.fairValue || 0).toLocaleString()
                      : `$${(item.fairValue || 0).toFixed(2)}`}
                  </span>
                  <span className="text-[10px] text-zinc-400 font-mono uppercase tracking-widest font-bold">
                    Estimated FV
                  </span>
                </div>
                <div className="grid grid-cols-1 gap-1.5">
                  {(item.assumptions || []).map((ass: any, i: number) => (
                    <div key={i} className="flex items-center justify-between text-[10px] font-mono">
                      <span className="text-zinc-500">{ass.label}</span>
                      <span className="text-zinc-300 font-bold">{ass.value}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Monte Carlo Probability Area */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50`}>
          <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
            <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
              <BarChart3 className="w-3.5 h-3.5" /> Monte Carlo Simulation
            </CardTitle>
          </CardHeader>
          <CardContent className="p-5 space-y-8 max-h-[500px] overflow-y-auto custom-scrollbar">
            {(valuation.monteCarlo || []).map((sim: any, idx: number) => (
              <div
                key={idx}
                className="space-y-6 pb-6 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0 animate-in fade-in slide-in-from-bottom-2 duration-300"
              >
                <div className="flex items-center justify-between">
                  <h4 className="text-xs font-bold font-mono text-zinc-400 uppercase tracking-tighter">
                    <span className={`text-${accentColor}-500 mr-2`}>▶</span>
                    {sim.symbol} Probabilities
                  </h4>
                  <span className="text-[9px] text-zinc-500 font-mono">
                    n={(sim.iterations || 0).toLocaleString()} PATHS
                  </span>
                </div>
                <div className="relative h-20 flex items-center justify-between px-2">
                  <div className="absolute inset-x-0 h-1 bg-gradient-to-r from-rose-500/20 via-emerald-500/30 to-rose-500/20 rounded-full" />
                  <div className="relative flex flex-col items-center">
                    <span className="text-[8px] font-bold text-rose-500 mb-3 bg-rose-500/10 px-1 rounded uppercase">
                      P10
                    </span>
                    <div className="w-1.5 h-1.5 rounded-full bg-rose-500 border border-white dark:border-zinc-900 shadow-lg" />
                    <span className="mt-1.5 font-mono text-[9px] font-bold">{(sim.p10 || 0).toLocaleString()}</span>
                  </div>
                  <div className="relative flex flex-col items-center">
                    <span className="text-[8px] font-bold text-emerald-500 mb-3 bg-emerald-500/10 px-1 rounded uppercase">
                      P50
                    </span>
                    <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-white dark:border-zinc-900 shadow-lg" />
                    <span className="mt-1.5 font-mono text-[10px] font-bold">{(sim.p50 || 0).toLocaleString()}</span>
                  </div>
                  <div className="relative flex flex-col items-center">
                    <span className="text-[8px] font-bold text-indigo-500 mb-3 bg-indigo-500/10 px-1 rounded uppercase">
                      P90
                    </span>
                    <div className="w-1.5 h-1.5 rounded-full bg-indigo-500 border border-white dark:border-zinc-900 shadow-lg" />
                    <span className="mt-1.5 font-mono text-[9px] font-bold">{(sim.p90 || 0).toLocaleString()}</span>
                  </div>
                </div>
              </div>
            ))}
            {(valuation.monteCarlo || []).length > 0 && (
              <div className="pt-2 text-[10px] text-zinc-500 leading-relaxed italic text-center p-3 rounded-lg bg-zinc-50 dark:bg-zinc-800/20 border border-dashed border-zinc-200 dark:border-zinc-700/50">
                Proprietary simulation showing range of probable outcomes over 10,000 algorithmic iterations.
              </div>
            )}
          </CardContent>
        </Card>

        {/* Sector Comparison */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 overflow-hidden`}>
          <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
            <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
              <PieChart className="w-3.5 h-3.5" /> Sector Benchmarking
            </CardTitle>
          </CardHeader>
          <CardContent className="p-0 max-h-[500px] overflow-y-auto custom-scrollbar">
            <div className="p-5 space-y-8">
              {(valuation.sectorComparison || []).map((sec: any, idx: number) => (
                <div
                  key={idx}
                  className="space-y-4 pb-6 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0 animate-in fade-in slide-in-from-bottom-2 duration-300"
                >
                  <div className="flex items-center justify-between">
                    <div className="text-xs font-bold font-mono uppercase text-zinc-400 tracking-tighter">
                      <span className={`text-${accentColor}-500 mr-2`}>▶</span>
                      {sec.symbol} vs. Peers
                    </div>
                    <Badge
                      variant="outline"
                      className="text-[8px] uppercase font-mono border-zinc-200 dark:border-zinc-800 scale-90"
                    >
                      {sec.sector}
                    </Badge>
                  </div>
                  <div className="space-y-3 pt-2">
                    {(sec.metrics || []).map((m: any, i: number) => (
                      <div key={i} className="space-y-1.5">
                        <div className="flex items-center justify-between text-[9px] uppercase font-bold tracking-tight">
                          <span className="text-zinc-500">{m.label}</span>
                          <span className={m.asset < m.avg ? 'text-emerald-500' : 'text-zinc-500'}>
                            {m.asset} <span className="opacity-50 mx-1">/</span> {m.avg}
                          </span>
                        </div>
                        <div className="relative h-1.5 w-full bg-zinc-100 dark:bg-zinc-800 rounded-full overflow-hidden">
                          <div
                            className="absolute inset-y-0 left-0 bg-zinc-400 opacity-20"
                            style={{ width: `${(Math.min(m.avg, 100) / (Math.max(m.asset, m.avg, 1) * 1.2)) * 100}%` }}
                          />
                          <div
                            className={`absolute inset-y-0 left-0 ${market === 'US' ? 'bg-indigo-500' : 'bg-emerald-500'}`}
                            style={{
                              width: `${(Math.min(m.asset, 100) / (Math.max(m.asset, m.avg, 1) * 1.2)) * 100}%`,
                            }}
                          />
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
