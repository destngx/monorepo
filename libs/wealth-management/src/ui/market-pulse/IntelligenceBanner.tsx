'use client';

import { Shield, TrendingUp } from 'lucide-react';
import { Badge } from '../badge';
import { GLASS_CARD, TERMINAL_FONT } from './constants';

export function IntelligenceBanner({
  scenarios,
  capitalFlow,
  market,
  timeframe,
}: {
  scenarios?: any[];
  capitalFlow?: any;
  market: string;
  timeframe: string;
}) {
  if (!scenarios || scenarios.length === 0 || !capitalFlow) return null;

  return (
    <div
      className={`${GLASS_CARD} flex flex-col divide-y divide-zinc-200 dark:divide-zinc-800/50 p-0 mb-8 rounded-2xl overflow-hidden`}
    >
      {/* 3 Scenario Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-zinc-200 dark:divide-zinc-800/50">
        {scenarios.map((scenario, idx) => {
          const regimeColor =
            scenario.regime === 'Risk-ON'
              ? 'text-emerald-400'
              : scenario.regime === 'Crisis'
                ? 'text-rose-400'
                : scenario.regime === 'Goldilocks'
                  ? 'text-indigo-400'
                  : scenario.regime === 'Stagflation'
                    ? 'text-amber-400'
                    : 'text-zinc-400';

          return (
            <div
              key={idx}
              className="p-5 space-y-3 relative group hover:bg-zinc-50/50 dark:hover:bg-zinc-900/20 transition-colors"
            >
              <div className="flex items-center justify-between">
                <span className="text-[9px] font-bold tracking-widest text-zinc-500 uppercase italic">
                  Scenario {idx + 1}
                </span>
                <Badge
                  variant="outline"
                  className={`text-[9px] font-mono border-none scale-90 ${
                    scenario.confidence >= 80
                      ? 'bg-emerald-500/10 text-emerald-500'
                      : scenario.confidence >= 50
                        ? 'bg-amber-500/10 text-amber-500'
                        : 'bg-rose-500/10 text-rose-500'
                  }`}
                >
                  {scenario.confidence}% CONF
                </Badge>
              </div>
              <div className="space-y-1">
                <h3 className={`text-sm font-bold tracking-tight uppercase ${regimeColor}`}>{scenario.name}</h3>
                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed">
                  {scenario.summaryEn}
                </p>
                {scenario.actionEn && (
                  <p className="text-xs font-semibold text-indigo-600 dark:text-indigo-400 leading-relaxed mt-2 p-2 bg-indigo-50 dark:bg-indigo-950/30 rounded-md border border-indigo-100 dark:border-indigo-900/50">
                    <span className="text-[9px] uppercase font-bold text-indigo-700 dark:text-indigo-500 block mb-1">Reaction Plan</span>
                    {scenario.actionEn}
                  </p>
                )}
              </div>
              <div className="space-y-2 border-l pl-2 border-zinc-200 dark:border-zinc-800/50">
                <p className="text-[10px] text-zinc-400 dark:text-zinc-500 italic font-mono">
                  {scenario.summaryVi}
                </p>
                {scenario.actionVi && (
                  <p className="text-[10px] font-semibold text-indigo-600 dark:text-indigo-400 italic font-mono p-1.5 bg-indigo-50/50 dark:bg-indigo-950/20 rounded-md border border-indigo-100/50 dark:border-indigo-900/30">
                    Hành động: {scenario.actionVi}
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {/* Smart Money Flow Section */}
      <div className="p-5 flex flex-col md:flex-row items-center gap-6 bg-zinc-50/50 dark:bg-zinc-950/40">
        <div className="flex items-center gap-4 min-w-[200px]">
          <div className="relative">
            <Shield className="w-8 h-8 text-zinc-200 dark:text-zinc-800" />
            <div className={`absolute inset-0 flex items-center justify-center`}>
              {capitalFlow.signal === 'RISK-ON' ? (
                <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse ring-4 ring-emerald-500/20" />
              ) : capitalFlow.signal === 'DEFENSIVE' ? (
                <div className="w-2.5 h-2.5 rounded-full bg-rose-500 ring-4 ring-rose-500/20" />
              ) : (
                <div className="w-2.5 h-2.5 rounded-full bg-amber-500 ring-4 ring-amber-500/20" />
              )}
            </div>
          </div>
          <div>
            <div className="text-[10px] font-bold tracking-[0.2em] text-zinc-500 uppercase font-mono mb-0.5">
              Smart Money Flow
            </div>
            <div className={`text-xl font-black tracking-tighter ${TERMINAL_FONT}`}>{capitalFlow.signal}</div>
          </div>
        </div>

        <div className="flex-1 space-y-1">
          <p className="text-xs text-zinc-600 dark:text-zinc-300 font-medium leading-relaxed">
            {capitalFlow.summaryEn}
          </p>
          <p className="text-[10px] text-zinc-500 italic font-mono">{capitalFlow.summaryVi}</p>
        </div>

        <div className="hidden lg:flex items-center gap-2 px-4 py-2 rounded-xl bg-white/50 dark:bg-zinc-900/50 border border-zinc-200/50 dark:border-zinc-800/50">
          <TrendingUp
            className={`w-3.5 h-3.5 ${capitalFlow.signal === 'RISK-ON' ? 'text-emerald-500' : 'text-zinc-400'}`}
          />
          <span className="text-[9px] font-bold font-mono text-zinc-500 uppercase tracking-tighter">
            Institutional Tracking Active
          </span>
        </div>
      </div>
    </div>
  );
}
