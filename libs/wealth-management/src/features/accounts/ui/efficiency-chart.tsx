"use client";

import { useState, useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@wealth-management/ui";
import { MaskedBalance } from "@wealth-management/ui";
import { 
  ComposedChart, 
  Bar, 
  Area,
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  Legend, 
  ResponsiveContainer,
  ReferenceLine,
  Cell
} from 'recharts';
import { CalendarRange, Filter, Trophy, Target, AlertCircle } from 'lucide-react';

interface ChartProps {
  cardStats: any[];
}

export function EfficiencyChart({ cardStats }: ChartProps) {
  const availableYears = useMemo(() => {
    const years = new Set<string>();
    cardStats.forEach(card => {
      card.history.forEach((h: any) => {
        years.add(h.month.split('-')[0]);
      });
    });
    return Array.from(years).sort().reverse();
  }, [cardStats]);

  const [selectedYear, setSelectedYear] = useState<string>(availableYears[0] || new Date().getFullYear().toString());
  const [selectedCard, setSelectedCard] = useState<string>('All');

  const chartData = useMemo(() => {
    const months = ['01', '02', '03', '04', '05', '06', '07', '08', '09', '10', '11', '12'];
    let runningTotalYtdPotential = 0;
    let runningTotalYtdEarned = 0;

    return months.map((monthNum) => {
      const monthKey = `${selectedYear}-${monthNum}`;
      const dataPoint: any = {
        name: new Date(`${selectedYear}-${monthNum}-01`).toLocaleDateString('en-US', { month: 'short' }),
        monthKey,
      };

      const accountMonthlyPotential: Record<string, number> = {};
      const accountMonthlyEarned: Record<string, number> = {};

      cardStats.forEach(card => {
        const mData = card.history.find((h: any) => h.month === monthKey);
        const pot = mData ? mData.cashback : 0; 
        const act = mData ? mData.actualRefund : 0;
        
        const accKey = card.accountKey || 'default';
        accountMonthlyPotential[accKey] = (accountMonthlyPotential[accKey] || 0) + pot;
        accountMonthlyEarned[accKey] = (accountMonthlyEarned[accKey] || 0) + act;

        if (selectedCard === card.name) {
          dataPoint.earned = act;
          dataPoint.missed = Math.max(0, Math.min(pot, 600000) - act);
        }
      });

      let combinedMonthlyCappedPot = 0;
      let combinedMonthlyEarned = 0;

      Object.keys(accountMonthlyPotential).forEach(accKey => {
         combinedMonthlyCappedPot += Math.min(accountMonthlyPotential[accKey], 600000);
         combinedMonthlyEarned += accountMonthlyEarned[accKey];
      });

      if (selectedCard === 'All') {
        dataPoint.earned = combinedMonthlyEarned;
        dataPoint.missed = Math.max(0, combinedMonthlyCappedPot - combinedMonthlyEarned);
      }

      runningTotalYtdPotential += combinedMonthlyCappedPot;
      runningTotalYtdEarned += combinedMonthlyEarned;

      dataPoint.ytdEarned = runningTotalYtdEarned;
      dataPoint.ytdPotential = runningTotalYtdPotential;

      return dataPoint;
    });
  }, [cardStats, selectedYear, selectedCard]);

  const totals = useMemo(() => {
    return chartData.reduce((acc, curr) => ({
      earned: acc.earned + curr.earned,
      missed: acc.missed + curr.missed,
      ytdEarned: curr.ytdEarned,
      ytdPotential: curr.ytdPotential,
    }), { earned: 0, missed: 0, ytdEarned: 0, ytdPotential: 0 });
  }, [chartData]);

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-zinc-950/95 border border-zinc-800 p-4 rounded-xl shadow-2xl backdrop-blur-md min-w-[240px] space-y-3">
          <div className="flex justify-between items-center pb-2 border-b border-zinc-800">
            <span className="font-bold text-sm text-zinc-100">{label} {selectedYear}</span>
            <span className="text-[10px] font-mono text-zinc-500 uppercase">Analysis</span>
          </div>
          <div className="space-y-2">
            {payload.map((entry: any, index: number) => (
              <div key={index} className="flex justify-between items-center gap-4">
                <div className="flex items-center gap-2">
                  <div className="w-1.5 h-1.5 rounded-full" style={{ backgroundColor: entry.color }} />
                  <span className="text-xs text-zinc-400 capitalize">{entry.name}</span>
                </div>
                <span className="text-xs font-mono font-bold text-zinc-100">
                  <MaskedBalance amount={entry.value} />
                </span>
              </div>
            ))}
          </div>
          {payload[0] && payload[1] && (
            <div className="pt-2 border-t border-zinc-800/50 mt-1">
               <div className="flex justify-between items-center italic text-[10px]">
                  <span className="text-zinc-500">Efficiency</span>
                  <span className="text-emerald-500 font-bold">
                    {Math.round((payload[0].value / (payload[0].value + payload[1].value)) * 100)}%
                  </span>
               </div>
            </div>
          )}
        </div>
      );
    }
    return null;
  };

  return (
    <Card className="border-none bg-white dark:bg-zinc-950 shadow-xl shadow-zinc-200/50 dark:shadow-none overflow-hidden">
      <CardHeader className="flex flex-col sm:flex-row sm:items-center justify-between gap-6 pb-8">
         <div className="space-y-1">
            <div className="flex items-center gap-2">
               <div className="p-2 bg-indigo-500/10 rounded-lg">
                  <CalendarRange className="h-5 w-5 text-indigo-500" />
               </div>
               <CardTitle className="text-xl font-black tracking-tight">Annual Cash Flow Analysis</CardTitle>
            </div>
            <CardDescription className="text-zinc-500 font-medium ml-11">Realized rewards vs theoretical potential at 600k cap.</CardDescription>
         </div>
         <div className="flex items-center gap-3">
            <div className="flex items-center gap-2 bg-zinc-100 dark:bg-zinc-900 rounded-xl p-1.5 border border-zinc-200 dark:border-zinc-800">
               <Filter className="h-3.5 w-3.5 text-zinc-400 ml-2" />
               <select 
                 className="bg-transparent text-xs border-none focus:ring-0 outline-none text-zinc-800 dark:text-zinc-200 py-1 font-bold pr-2 cursor-pointer"
                 value={selectedCard}
                 onChange={(e) => setSelectedCard(e.target.value)}
               >
                 <option value="All">Collective Intelligence</option>
                 {cardStats.map(c => (
                   <option key={c.name} value={c.name}>{c.name}</option>
                 ))}
               </select>
            </div>
            <select 
              className="bg-white dark:bg-zinc-950 border border-zinc-200 dark:border-zinc-800 text-xs rounded-xl px-4 py-2 focus:ring-2 focus:ring-indigo-500 font-black cursor-pointer shadow-sm"
              value={selectedYear}
              onChange={(e) => setSelectedYear(e.target.value)}
            >
              {availableYears.map(year => (
                <option key={year} value={year}>{year}</option>
              ))}
            </select>
         </div>
      </CardHeader>
      
      <CardContent className="space-y-8">
        {/* Quick Stats Grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-4">
           <div className="p-4 rounded-2xl bg-emerald-50/50 dark:bg-emerald-950/10 border border-emerald-100/50 dark:border-emerald-900/20 space-y-1">
              <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400">
                 <Trophy className="h-3.5 w-3.5" />
                 <span className="text-[10px] font-black uppercase tracking-widest">Total Earned</span>
              </div>
              <p className="text-2xl font-black tracking-tighter text-emerald-700 dark:text-emerald-300">
                 <MaskedBalance amount={totals.earned} />
              </p>
           </div>
           <div className="p-4 rounded-2xl bg-rose-50/50 dark:bg-rose-950/10 border border-rose-100/50 dark:border-rose-900/20 space-y-1">
              <div className="flex items-center gap-2 text-rose-600 dark:text-rose-400">
                 <AlertCircle className="h-3.5 w-3.5" />
                 <span className="text-[10px] font-black uppercase tracking-widest">Missed Opportunities</span>
              </div>
              <p className="text-2xl font-black tracking-tighter text-rose-700 dark:text-rose-300">
                 <MaskedBalance amount={totals.missed} />
              </p>
           </div>
           <div className="p-4 rounded-2xl bg-indigo-50/50 dark:bg-indigo-950/10 border border-indigo-100/50 dark:border-indigo-900/20 space-y-1">
              <div className="flex items-center gap-2 text-indigo-600 dark:text-indigo-400">
                 <Target className="h-3.5 w-3.5" />
                 <span className="text-[10px] font-black uppercase tracking-widest">Efficiency Rate</span>
              </div>
              <p className="text-2xl font-black tracking-tighter text-indigo-700 dark:text-indigo-300">
                 {totals.earned + totals.missed > 0 ? Math.round((totals.earned / (totals.earned + totals.missed)) * 100) : 0}%
              </p>
           </div>
        </div>

        <div className="w-full h-[450px] relative group">
          <ResponsiveContainer width="100%" height="100%">
            <ComposedChart
              data={chartData}
              margin={{ top: 20, right: 20, bottom: 20, left: 20 }}
            >
              <defs>
                <linearGradient id="gradEarned" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#10b981" stopOpacity={0.8}/>
                  <stop offset="100%" stopColor="#059669" stopOpacity={1}/>
                </linearGradient>
                <linearGradient id="gradMissed" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#f43f5e" stopOpacity={0.2}/>
                  <stop offset="100%" stopColor="#e11d48" stopOpacity={0.4}/>
                </linearGradient>
                <linearGradient id="gradYtd" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor="#6366f1" stopOpacity={0.2}/>
                  <stop offset="100%" stopColor="#6366f1" stopOpacity={0}/>
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="currentColor" className="text-zinc-200 dark:text-zinc-800" opacity={0.5} />
              <XAxis 
                dataKey="name" 
                axisLine={false}
                tickLine={false}
                tick={{ fill: 'currentColor', fontSize: 11, fontWeight: 'bold' }}
                className="text-zinc-400"
                dy={15}
              />
              <YAxis 
                yAxisId="left"
                axisLine={false}
                tickLine={false}
                tick={{ fill: 'currentColor', fontSize: 10, fontWeight: 'bold' }}
                className="text-zinc-400"
                tickFormatter={(value) => value >= 1000 ? `${(value / 1000).toFixed(0)}k` : value}
                dx={-10}
              />
              <YAxis 
                yAxisId="cumulative"
                orientation="right"
                axisLine={false}
                tickLine={false}
                tick={{ fill: 'currentColor', fontSize: 10, fontWeight: 'black' }}
                className="text-indigo-500"
                tickFormatter={(value) => value >= 1000000 ? `${(value / 1000000).toFixed(1)}M` : `${(value / 1000).toFixed(0)}k`}
                dx={10}
              />
              <Tooltip content={<CustomTooltip />} cursor={{ fill: 'currentColor', className: 'text-zinc-100 dark:text-zinc-900', opacity: 0.1 }} />
              
              <ReferenceLine 
                yAxisId="left" 
                y={600000} 
                stroke="#6366f1" 
                strokeDasharray="8 8" 
                strokeWidth={1.5}
                label={{ 
                  value: '600k CAP', 
                  position: 'insideTopRight', 
                  fill: '#6366f1', 
                  fontSize: 9, 
                  fontWeight: 'black',
                  letterSpacing: '0.1em'
                }} 
              />

              <Area
                yAxisId="cumulative"
                type="monotone"
                dataKey="ytdEarned"
                name="YTD Earned"
                stroke="#6366f1"
                strokeWidth={3}
                fill="url(#gradYtd)"
                animationDuration={2000}
              />

              <Bar 
                yAxisId="left" 
                dataKey="earned" 
                name="Earned" 
                stackId="a" 
                fill="url(#gradEarned)" 
                barSize={32}
              />
              <Bar 
                yAxisId="left" 
                dataKey="missed" 
                name="Missed" 
                stackId="a" 
                fill="url(#gradMissed)" 
                radius={[6, 6, 0, 0]}
                barSize={32}
              />

              <Legend 
                verticalAlign="top" 
                align="right"
                height={40} 
                iconType="circle"
                wrapperStyle={{ fontSize: '10px', fontWeight: 'bold', textTransform: 'uppercase', letterSpacing: '0.1em', paddingTop: '0' }}
                formatter={(value) => <span className="text-zinc-500">{value}</span>}
              />
            </ComposedChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
