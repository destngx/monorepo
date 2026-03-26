'use client';

import { useTheme } from 'next-themes';
import { Info } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '../card';
import { Button } from '../button';
import { Tooltip as ShadcnTooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../tooltip';
import { GLASS_CARD } from './constants';

export function CorrelationHeatmap({ title, assets, matrix }: { title: string; assets: string[]; matrix: number[][] }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  if (!assets.length || !matrix.length) return null;

  const getColor = (val: number) => {
    if (val > 0) return `rgba(16, 185, 129, ${val * (isDark ? 0.7 : 0.6)})`;
    if (val < 0) return `rgba(244, 63, 94, ${Math.abs(val) * (isDark ? 0.7 : 0.6)})`;
    return isDark ? 'rgba(24, 24, 27, 0.5)' : 'rgba(244, 244, 245, 0.8)';
  };

  return (
    <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/30`}>
      <CardHeader className="bg-zinc-50 dark:bg-zinc-950/40 border-b border-zinc-200 dark:border-zinc-800/50 py-3 px-4 flex flex-row items-center justify-between">
        <CardTitle className="text-[9px] uppercase font-bold tracking-[0.3em] text-zinc-400 dark:text-zinc-500 font-mono">
          {title}
        </CardTitle>
        <TooltipProvider>
          <ShadcnTooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" className="h-4 w-4 p-0 text-zinc-400 hover:text-indigo-400">
                <Info className="w-3 h-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent
              side="left"
              className="w-64 p-3 bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 shadow-2xl"
            >
              <div className="space-y-3">
                <div className="space-y-1">
                  <h4 className="text-[10px] font-bold uppercase text-indigo-500">How to Read</h4>
                  <p className="text-[11px] leading-relaxed text-zinc-600 dark:text-zinc-400">
                    Positive correlation means assets move together; negative means opposite. A broken correlation alert
                    signals a market regime shift.
                  </p>
                </div>
                <div className="space-y-1 border-t border-zinc-100 dark:border-zinc-800/50 pt-2 text-zinc-500 italic">
                  <h4 className="text-[10px] font-bold uppercase text-emerald-500">Cách đọc</h4>
                  <p className="text-[11px] leading-relaxed">
                    Tương quan dương (+) là cùng chiều, âm (-) là ngược chiều. Cảnh báo "broken correlation" là dấu hiệu
                    chuyển đổi trạng thái thị trường.
                  </p>
                </div>
              </div>
            </TooltipContent>
          </ShadcnTooltip>
        </TooltipProvider>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr>
                <th className="p-2 border-b border-r border-zinc-200 dark:border-zinc-800/50 bg-zinc-100 dark:bg-zinc-950/60"></th>
                {assets.map((a, i) => (
                  <th
                    key={i}
                    className="p-2 border-b border-zinc-200 dark:border-zinc-800/50 text-[8px] font-mono text-zinc-400 dark:text-zinc-600 text-center uppercase tracking-tighter w-10 min-w-[36px]"
                  >
                    {a.slice(0, 5)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {matrix.map((row, i) => (
                <tr key={i} className="group/row hover:bg-black/[0.02] dark:hover:bg-white/[0.03] transition-colors">
                  <td className="p-2 border-r border-zinc-200 dark:border-zinc-800/50 text-[8px] font-bold uppercase bg-zinc-100 dark:bg-zinc-950/60 text-zinc-500 group-hover/row:text-zinc-900 dark:group-hover/row:text-zinc-200 transition-colors">
                    {assets[i]}
                  </td>
                  {row.map((val, j) => (
                    <TooltipProvider key={j}>
                      <ShadcnTooltip>
                        <TooltipTrigger asChild>
                          <td
                            className="p-1 text-center text-[9px] font-mono border border-zinc-200 dark:border-zinc-900/40 cursor-crosshair transition-all hover:brightness-110 active:scale-95"
                            style={{
                              backgroundColor: getColor(val),
                              color:
                                Math.abs(val) > 0.6
                                  ? isDark
                                    ? 'white'
                                    : '#000'
                                  : isDark
                                    ? 'rgba(255,255,255,0.4)'
                                    : 'rgba(0,0,0,0.5)',
                            }}
                          >
                            {val === 1 ? '1.0' : val.toFixed(2)}
                          </td>
                        </TooltipTrigger>
                        <TooltipContent
                          side="top"
                          className="bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 text-[10px] font-mono p-2 rounded-lg shadow-2xl"
                        >
                          <div className="font-bold text-zinc-900 dark:text-zinc-300">
                            {assets[i]} × {assets[j]}
                          </div>
                          <div
                            className={`mt-0.5 ${val > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'}`}
                          >
                            COEFF: {val.toFixed(3)}
                          </div>
                        </TooltipContent>
                      </ShadcnTooltip>
                    </TooltipProvider>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}
