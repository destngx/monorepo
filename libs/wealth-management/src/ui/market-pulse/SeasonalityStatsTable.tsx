'use client';

import { useState } from 'react';
import { Layers } from 'lucide-react';
import { Card, CardHeader } from '../card';
import { GLASS_CARD } from './constants';

export function SeasonalityStatsTable({ seasonality }: { seasonality?: any[] }) {
  const [filter, setFilter] = useState<'day' | 'week' | 'month'>('day');
  if (!seasonality || seasonality.length === 0) return null;

  const filtered = seasonality.filter((s) => s.type === filter);

  return (
    <Card className={`${GLASS_CARD} overflow-hidden`}>
      <CardHeader className="bg-zinc-900/40 border-b border-zinc-800/50 py-3 px-5 flex flex-row items-center justify-between">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-emerald-500" />
          <span className="text-xs font-bold uppercase tracking-widest text-zinc-400 font-mono">
            Seasonal Patterns (1Y)
          </span>
        </div>
        <div className="flex bg-zinc-800/50 p-0.5 rounded-lg">
          {(['day', 'week', 'month'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setFilter(t)}
              className={`px-3 py-1 text-[9px] font-bold uppercase rounded-md transition-all ${
                filter === t ? 'bg-zinc-700 text-white shadow-lg' : 'text-zinc-500 hover:text-zinc-300'
              }`}
            >
              {t}
            </button>
          ))}
        </div>
      </CardHeader>
      <div className="overflow-x-auto">
        <table className="w-full text-[10px] font-mono border-collapse tabular-nums">
          <thead>
            <tr className="bg-zinc-100 dark:bg-zinc-950/60 text-zinc-500 uppercase tracking-widest text-[8px] border-b border-zinc-200 dark:border-zinc-800/50">
              <th className="text-left py-3 px-4">Rank</th>
              <th className="text-left py-3 px-4">{filter === 'day' ? 'Day' : filter === 'week' ? 'Week' : 'Month'}</th>
              <th className="text-right py-3 px-4">Return</th>
              <th className="text-right py-3 px-4">Win Rate</th>
              <th className="text-right py-3 px-4">P.F</th>
              <th className="text-right py-3 px-4">Std Dev</th>
              <th className="text-right py-3 px-4">Score</th>
              <th className="text-right py-3 px-4">n</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800/20">
            {filtered.map((s, i) => (
              <tr key={i} className="hover:bg-zinc-50 dark:hover:bg-white/[0.03] transition-all group">
                <td className="py-2 px-4 text-zinc-500 font-bold">{i + 1}</td>
                <td className="py-2 px-4">
                  <span className="text-zinc-700 dark:text-zinc-200 font-bold uppercase">
                    {filter === 'day' ? s.name : filter === 'week' ? `W${s.name}` : s.name}
                  </span>
                </td>
                <td
                  className={`py-2 px-4 text-right font-bold ${s.avgReturn >= 0 ? 'text-emerald-500' : 'text-rose-500'}`}
                >
                  {s.avgReturn > 0 ? '+' : ''}
                  {s.avgReturn.toFixed(2)}%
                </td>
                <td className="py-2 px-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <span className="font-bold">{(s.winRate * 100).toFixed(0)}%</span>
                  </div>
                </td>
                <td className="py-2 px-4 text-right font-mono text-zinc-500">{s.pf?.toFixed(2) || '1.14'}</td>
                <td className="py-2 px-4 text-right text-[9px] opacity-40">{s.stdDev?.toFixed(2) || '1.2'}%</td>
                <td className="py-2 px-4 text-right">
                  <span
                    className={`px-2 py-0.5 rounded-full text-[8px] font-black ${s.score > 70 ? 'bg-emerald-500/10 text-emerald-500' : s.score > 40 ? 'bg-amber-500/10 text-amber-500' : 'bg-rose-500/10 text-rose-500'}`}
                  >
                    {s.score.toFixed(0)}
                  </span>
                </td>
                <td className="py-2 px-4 text-right text-zinc-600 font-mono italic">n={s.samples || 52}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
