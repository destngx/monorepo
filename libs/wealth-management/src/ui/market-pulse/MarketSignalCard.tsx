'use client';

import { ArrowUpRight, ArrowDownRight, ArrowRight, AlertCircle, Target, Moon, Shield, TrendingUp } from 'lucide-react';
import { Card, CardHeader, CardContent } from '../card';
import { Badge } from '../badge';
import { GLASS_CARD } from './constants';

export function MarketSignalCard({ signals, market }: { signals?: any; market: string }) {
  if (!signals) return null;

  const action = signals.action || 'HOLD/WATCH';

  const config: Record<string, { color: string; icon: any; bg: string; text: string; border: string }> = {
    LONG: {
      color: 'emerald',
      icon: ArrowUpRight,
      bg: 'bg-emerald-500/10',
      text: 'text-emerald-500',
      border: 'border-emerald-500/20',
    },
    SHORT: {
      color: 'rose',
      icon: ArrowDownRight,
      bg: 'bg-rose-500/10',
      text: 'text-rose-500',
      border: 'border-rose-500/20',
    },
    EXIT: {
      color: 'rose',
      icon: ArrowRight,
      bg: 'bg-rose-500/10',
      text: 'text-rose-500',
      border: 'border-rose-500/30',
    },
    REDUCE: {
      color: 'amber',
      icon: AlertCircle,
      bg: 'bg-amber-500/10',
      text: 'text-amber-500',
      border: 'border-amber-500/30',
    },
    'TAKE PROFIT': {
      color: 'indigo',
      icon: Target,
      bg: 'bg-indigo-500/10',
      text: 'text-indigo-500',
      border: 'border-indigo-500/30',
    },
    'HOLD/WATCH': {
      color: 'zinc',
      icon: Moon,
      bg: 'bg-zinc-500/10',
      text: 'text-zinc-500',
      border: 'border-zinc-500/30',
    },
    AVOID: { color: 'zinc', icon: Shield, bg: 'bg-zinc-500/10', text: 'text-zinc-500', border: 'border-zinc-500/30' },
  };

  const current = config[action] || config['HOLD/WATCH'];
  const Icon = current.icon;

  return (
    <Card className={`${GLASS_CARD} ${current.border} shadow-2xl`}>
      <CardHeader
        className={`pb-4 border-b ${current.border} ${current.bg} flex flex-row items-center justify-between`}
      >
        <div className="flex items-center gap-3">
          <div className={`p-2 rounded-xl bg-white dark:bg-zinc-950 shadow-inner ${current.text}`}>
            <Icon className="w-5 h-5 animate-pulse" />
          </div>
          <div>
            <h3 className={`text-xl font-black tracking-tighter uppercase ${current.text}`}>{signals.action} SIGNAL</h3>
            <p className="text-[10px] uppercase font-mono text-zinc-500 tracking-widest font-bold">
              Institutional Action Required
            </p>
          </div>
        </div>
        <div className="flex flex-col items-end">
          <Badge className={`${current.bg} ${current.text} border-none font-mono text-[10px] font-black px-3`}>
            {signals.confidence}% CONFIDENCE
          </Badge>
          <div className="text-[9px] uppercase font-bold text-zinc-500 mt-1">{market} REGIME</div>
        </div>
      </CardHeader>

      <CardContent className="p-6 relative">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
          <div className="space-y-4">
            <div className="space-y-1">
              <span className="text-[10px] font-bold text-zinc-500 uppercase flex items-center gap-1.5">
                <Target className="w-3 h-3" /> Trading Thesis
              </span>
              <p className="text-sm font-bold leading-relaxed">{signals.reasons[0]}</p>
              <p className="text-xs text-zinc-500 italic font-mono leading-relaxed">{signals.reasonsVi[0]}</p>
            </div>

            <div className="grid grid-cols-2 gap-4 pt-2">
              <div className="p-4 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800/50">
                <span className="text-[9px] font-bold text-zinc-500 uppercase block mb-1">Entry Range</span>
                <span className={`text-lg font-mono font-black ${current.text}`}>
                  {market === 'VN'
                    ? (signals.entry?.toLocaleString() ?? 'N/A')
                    : `$${signals.entry?.toFixed(2) ?? 'N/A'}`}
                </span>
              </div>
              <div className="p-4 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800/50">
                <span className="text-[9px] font-bold text-rose-500 uppercase block mb-1">Stop Loss</span>
                <span className="text-lg font-mono font-black text-rose-500">
                  {market === 'VN'
                    ? (signals.stopLoss?.toLocaleString() ?? 'N/A')
                    : `$${signals.stopLoss?.toFixed(2) ?? 'N/A'}`}
                </span>
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <span className="text-[10px] font-bold text-zinc-500 uppercase tracking-widest">
                Execution Confluence
              </span>
              <span className="text-[10px] font-mono font-bold text-indigo-500">
                R:R 1:{signals.rr?.toFixed(1) ?? '2.0'}
              </span>
            </div>

            <div className="space-y-2">
              {(signals.reasons || []).slice(1).map((r: string, i: number) => (
                <div
                  key={i}
                  className="flex items-start gap-3 p-3 rounded-xl bg-zinc-50 dark:bg-zinc-900/40 border border-zinc-200 dark:border-zinc-800/20 group hover:border-indigo-500/30 transition-colors"
                >
                  <div className="mt-0.5 p-1 rounded bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 group-hover:bg-emerald-500 group-hover:border-emerald-500 transition-colors">
                    <TrendingUp className="w-2.5 h-2.5 text-zinc-400 group-hover:text-white" />
                  </div>
                  <div>
                    <p className="text-[11px] font-medium leading-normal">{r}</p>
                    <p className="text-[9px] text-zinc-500 italic font-mono mt-0.5">{signals.reasonsVi?.[i + 1]}</p>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Dynamic Warning for AVOID/EXIT */}
        {(action === 'AVOID' || action === 'EXIT') && (
          <div className="mt-6 p-3 rounded-xl bg-rose-500/5 border border-rose-500/10 flex items-center gap-3">
            <AlertCircle className="w-4 h-4 text-rose-500 animate-pulse" />
            <p className="text-[10px] font-bold text-rose-400 uppercase tracking-tight">
              High volatility or distribution detected. Liquidating positions is recommended for risk mitigation.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
