'use client';

import { useMemo } from 'react';
import { AreaChart, Area, ResponsiveContainer, YAxis } from 'recharts';
import { subDays } from 'date-fns';
import { Transaction } from '@wealth-management/types';

interface AccountTrendSparklineProps {
  accountName: string;
  currentBalance: number;
  transactions: Transaction[];
}

export function AccountTrendSparkline({ accountName, currentBalance, transactions }: AccountTrendSparklineProps) {
  const sparklineData = useMemo((): Array<{ value: number }> => {
    const data: Array<{ value: number }> = [];
    let runningBalance = currentBalance;
    const now = new Date();

    // Filter transactions for this account
    const accountTxns = transactions.filter((t) => t.accountName === accountName);

    // Go backwards from today (60 days)
    for (let i = 0; i < 60; i++) {
      const targetDate = subDays(now, i);
      const dayTxns = accountTxns.filter((t) => {
        const d = new Date(t.date);
        return d.toDateString() === targetDate.toDateString();
      });

      const dayFlow = dayTxns.reduce((sum, t) => sum + (t.deposit || 0) - (t.payment || 0), 0);

      data.unshift({ value: runningBalance });
      runningBalance -= dayFlow;
    }
    return data;
  }, [accountName, currentBalance, transactions]);

  const firstValue = sparklineData[0]?.value || 0;
  const lastValue = sparklineData[sparklineData.length - 1]?.value || 0;
  const isPositive = lastValue >= firstValue;

  return (
    <div className="w-16 h-8 opacity-60 hover:opacity-100 transition-opacity">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={sparklineData}>
          <defs>
            <linearGradient id={`color-${accountName.replace(/\s+/g, '-')}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={isPositive ? '#10b981' : '#f43f5e'} stopOpacity={0.2} />
              <stop offset="95%" stopColor={isPositive ? '#10b981' : '#f43f5e'} stopOpacity={0} />
            </linearGradient>
          </defs>
          <YAxis hide domain={['dataMin - 1000', 'dataMax + 1000']} />
          <Area
            type="monotone"
            dataKey="value"
            stroke={isPositive ? '#10b981' : '#f43f5e'}
            strokeWidth={1.5}
            fill={`url(#color-${accountName.replace(/\s+/g, '-')})`}
            isAnimationActive={false}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}
