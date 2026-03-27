'use client';

import { Card, CardContent } from '@wealth-management/ui/card';
import { GoalProjection } from '../model/types';
import { TrendingUp, Clock, Rocket } from 'lucide-react';
import { formatVND } from '@wealth-management/utils';

interface AIInsightsProps {
  projection: GoalProjection;
}

export function AIGoalInsights({ projection }: AIInsightsProps) {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card className="border-none bg-muted/30">
          <CardContent className="p-4 flex items-start gap-3">
            <div className="p-2 bg-indigo-500/10 rounded-lg">
              <Clock className="h-4 w-4 text-indigo-500" />
            </div>
            <div className="space-y-1">
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Forecast</p>
              <p className="text-sm leading-snug">
                At your current pace of{' '}
                <span className="font-bold text-indigo-500">
                  {formatVND(projection.currentPace.monthlyContribution)}
                </span>
                /month, you'll reach your goal in{' '}
                <span className="font-bold">{projection.currentPace.completionDate}</span>.
              </p>
            </div>
          </CardContent>
        </Card>

        <Card className="border-none bg-muted/30">
          <CardContent className="p-4 flex items-start gap-3">
            <div className="p-2 bg-emerald-500/10 rounded-lg">
              <TrendingUp className="h-4 w-4 text-emerald-500" />
            </div>
            <div className="space-y-1">
              <p className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">Recommendation</p>
              <p className="text-sm leading-snug">
                To hit your target deadline, you need to save{' '}
                <span className="font-bold text-emerald-500">
                  {formatVND(projection.requiredPace.monthlyContribution)}
                </span>
                /month.
              </p>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card className="overflow-hidden border-indigo-500/20 bg-linear-to-r from-indigo-500/5 to-purple-500/5">
        <CardContent className="p-6">
          <div className="flex items-center gap-3 mb-4">
            <Rocket className="h-5 w-5 text-indigo-500" />
            <h4 className="font-bold text-sm uppercase tracking-tight">Scenario Comparison</h4>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-muted-foreground text-[10px] uppercase font-bold text-left border-b border-indigo-500/10">
                  <th className="pb-2">Scenario</th>
                  <th className="pb-2">Monthly</th>
                  <th className="pb-2">Completion</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-indigo-500/10">
                {projection.scenarios.map((s, i) => (
                  <tr key={i} className="group">
                    <td className="py-3 font-medium text-muted-foreground">{s.label}</td>
                    <td className="py-3 font-bold text-indigo-500">{formatVND(s.monthlyContribution)}</td>
                    <td className="py-3 font-medium">{s.completionDate}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
