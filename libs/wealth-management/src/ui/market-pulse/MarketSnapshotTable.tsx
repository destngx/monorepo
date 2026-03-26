'use client';

import { Flame, Moon, ArrowUpRight, ArrowDownRight } from 'lucide-react';
import { type MarketAsset } from '@wealth-management/services';

export function MarketSnapshotTable({ assets }: { assets: MarketAsset[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[11px] font-mono border-collapse tabular-nums">
        <thead>
          <tr className="bg-zinc-100 dark:bg-zinc-950/60 text-zinc-500 dark:text-zinc-500 uppercase tracking-widest text-[9px] border-b border-zinc-200 dark:border-zinc-800/50">
            <th className="text-left py-4 px-6 font-bold">INSTRUMENT</th>
            <th className="text-right py-4 px-4 font-bold">SIGNAL</th>
            <th className="text-right py-4 px-4 font-bold">Δ%</th>
            <th className="text-right py-4 px-4 font-bold">DAY</th>
            <th className="text-right py-4 px-4 font-bold">WEEK</th>
            <th className="text-center py-4 px-6 font-bold">STATUS</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800/20">
          {assets.map((a, i) => (
            <tr key={i} className="hover:bg-zinc-50 dark:hover:bg-white/[0.03] transition-all group">
              <td className="py-3 px-6 border-r border-zinc-200 dark:border-zinc-800/30">
                <div className="flex flex-col">
                  <span className="font-bold text-zinc-700 dark:text-zinc-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors uppercase">
                    {a.name}
                  </span>
                  <span className="text-[9px] text-zinc-400 dark:text-zinc-600 font-bold opacity-50">{a.symbol}</span>
                </div>
              </td>
              <td className="py-3 px-4 text-right font-medium text-zinc-600 dark:text-zinc-300">
                {a.market === 'VN' && a.symbol !== 'USD/VND'
                  ? new Intl.NumberFormat('vi-VN').format(a.price)
                  : new Intl.NumberFormat('en-US', {
                      style: 'currency',
                      currency: 'USD',
                      maximumFractionDigits: 2,
                    }).format(a.price)}
              </td>
              <td
                className={`py-3 px-4 text-right font-bold ${a.percentChange >= 0 ? 'text-emerald-400' : 'text-rose-400'}`}
              >
                <div className="flex items-center justify-end gap-1">
                  {a.percentChange > 0 ? '+' : ''}
                  {a.percentChange.toFixed(2)}
                  <span className="text-[9px] opacity-70">%</span>
                </div>
              </td>
              <td
                className={`py-3 px-4 text-right opacity-60 ${a.dayChange >= 0 ? 'text-emerald-500' : 'text-rose-500'}`}
              >
                {a.dayChange.toFixed(2)}%
              </td>
              <td
                className={`py-3 px-4 text-right opacity-40 ${a.weekChange >= 0 ? 'text-emerald-600' : 'text-rose-600'}`}
              >
                {a.weekChange.toFixed(2)}%
              </td>
              <td className="py-3 px-6 text-center">
                <div className="flex items-center justify-center gap-2">
                  {a.momentum === 'fire' ? (
                    <div className="p-1 rounded bg-orange-500/10">
                      <Flame className="w-3 h-3 text-orange-500 animate-pulse" />
                    </div>
                  ) : a.momentum === 'sleep' ? (
                    <Moon className="w-3 h-3 text-zinc-800" />
                  ) : a.direction === 'up' ? (
                    <ArrowUpRight className="w-3 h-3 text-emerald-500/50" />
                  ) : (
                    <ArrowDownRight className="w-3 h-3 text-rose-500/50" />
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
