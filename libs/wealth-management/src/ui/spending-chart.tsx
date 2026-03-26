'use client';

import { useState, useMemo } from 'react';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from './card';
import { Transaction } from '@wealth-management/types';
import { format, startOfYear, endOfMonth, eachMonthOfInterval, isSameMonth, startOfMonth, subMonths } from 'date-fns';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './select';

import { getEffectiveDate } from '@wealth-management/utils';

export function SpendingChart({ transactions }: { transactions: Transaction[] }) {
  const [range, setRange] = useState('ytd');

  const chartData = useMemo(() => {
    const now = new Date();
    let start: Date;
    const end: Date = endOfMonth(now);

    if (range === 'ytd') {
      start = startOfYear(now);
      // Let's make sure end is the full year so the chart scale looks correct if desired, or leave up to current month so as not to show empty upcoming months
    } else if (range === '6m') {
      start = startOfMonth(subMonths(now, 5));
    } else {
      // 12m
      start = startOfMonth(subMonths(now, 11));
    }

    const months = eachMonthOfInterval({ start, end });

    return months.map((d) => {
      const monthKey = format(d, 'MMM yy');

      const monthlyTxns = transactions.filter((t) => {
        const txDate = getEffectiveDate(t);
        return isSameMonth(txDate, d);
      });

      const income = monthlyTxns.reduce((sum, t) => {
        if (t.categoryType === 'income') return sum + (t.deposit || 0);
        return sum;
      }, 0);

      const expense = monthlyTxns.reduce((sum, t) => {
        if (t.categoryType === 'expense') return sum + (t.payment || 0);
        return sum;
      }, 0);

      return {
        name: monthKey,
        Income: income,
        Expense: expense,
      };
    });
  }, [transactions, range]);

  return (
    <Card className="shadow-sm overflow-hidden">
      <CardHeader className="flex flex-row items-center justify-between pb-4">
        <div>
          <CardTitle className="text-lg">Income vs Expense</CardTitle>
          <CardDescription>Cash flow performance over time</CardDescription>
        </div>
        <Select value={range} onValueChange={setRange}>
          <SelectTrigger className="w-[120px] h-8 text-xs">
            <SelectValue placeholder="Range" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="ytd">Year to Date</SelectItem>
            <SelectItem value="6m">Last 6 Months</SelectItem>
            <SelectItem value="12m">Last 12 Months</SelectItem>
          </SelectContent>
        </Select>
      </CardHeader>
      <CardContent className="p-0 overflow-x-auto">
        <div className="w-full h-80 min-w-[600px]">
          <ResponsiveContainer width="100%" height="100%" minWidth={0} minHeight={0}>
            <BarChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#e5e7eb" />
              <XAxis dataKey="name" fontSize={11} tickLine={false} axisLine={false} />
              <YAxis
                fontSize={11}
                tickLine={false}
                axisLine={false}
                tickFormatter={(value) => `${(value / 1000000).toFixed(1)}M`}
              />
              <Tooltip
                formatter={(value: any) =>
                  new Intl.NumberFormat('vi-VN', { style: 'currency', currency: 'VND' }).format(value)
                }
                cursor={{ fill: 'rgba(0,0,0,0.05)' }}
                contentStyle={{ fontSize: '12px', borderRadius: '8px' }}
              />
              <Legend
                verticalAlign="top"
                align="right"
                wrapperStyle={{ fontSize: '11px', paddingBottom: '20px' }}
                iconSize={10}
              />
              <Bar dataKey="Income" fill="#10b981" radius={[2, 2, 0, 0]} barSize={12} />
              <Bar dataKey="Expense" fill="#f43f5e" radius={[2, 2, 0, 0]} barSize={12} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </CardContent>
    </Card>
  );
}
