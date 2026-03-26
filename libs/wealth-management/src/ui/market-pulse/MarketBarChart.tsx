'use client';

import { useTheme } from 'next-themes';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
  ReferenceLine,
} from 'recharts';
import { type MarketAsset } from '@wealth-management/services';

export function MarketBarChart({ data }: { data: MarketAsset[] }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  if (!data.length)
    return (
      <div className="flex items-center justify-center h-full text-zinc-400 dark:text-zinc-600 font-mono text-[10px]">
        AWAITING STREAM...
      </div>
    );

  return (
    <ResponsiveContainer width="100%" height="100%" minWidth={0}>
      <BarChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="barUp" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#10b981" stopOpacity={0.8} />
            <stop offset="95%" stopColor="#10b981" stopOpacity={0.1} />
          </linearGradient>
          <linearGradient id="barDown" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#f43f5e" stopOpacity={0.8} />
            <stop offset="95%" stopColor="#f43f5e" stopOpacity={0.1} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={isDark ? '#18181b' : '#f4f4f5'} />
        <XAxis
          dataKey="name"
          stroke={isDark ? '#3f3f46' : '#d4d4d8'}
          fontSize={10}
          tickLine={false}
          axisLine={false}
          interval={0}
          tick={{ fill: isDark ? '#52525b' : '#a1a1aa', fontWeight: 600 }}
        />
        <YAxis
          stroke={isDark ? '#3f3f46' : '#d4d4d8'}
          fontSize={10}
          tickLine={false}
          axisLine={false}
          unit="%"
          tick={{ fill: isDark ? '#52525b' : '#a1a1aa' }}
        />
        <Tooltip
          contentStyle={{
            backgroundColor: isDark ? 'rgba(9, 9, 11, 0.95)' : 'rgba(255, 255, 255, 0.95)',
            backdropFilter: 'blur(12px)',
            border: isDark ? '1px solid rgba(63, 63, 70, 0.4)' : '1px solid rgba(228, 228, 231, 0.8)',
            borderRadius: '12px',
            fontSize: '10px',
            color: isDark ? '#fff' : '#18181b',
            boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.1)',
          }}
          cursor={{ fill: isDark ? 'rgba(255, 255, 255, 0.02)' : 'rgba(0, 0, 0, 0.02)' }}
          formatter={(value: any) => [`${Number(value) > 0 ? '+' : ''}${Number(value).toFixed(2)}%`, 'VELOCITY']}
        />
        <ReferenceLine y={0} stroke={isDark ? '#27272a' : '#e4e4e7'} strokeWidth={1} />
        <Bar dataKey="percentChange" radius={[4, 4, 0, 0]}>
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.percentChange >= 0 ? 'url(#barUp)' : 'url(#barDown)'} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}
