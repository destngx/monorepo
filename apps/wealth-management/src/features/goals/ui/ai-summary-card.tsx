'use client';

import { Card, CardContent } from '@/components/ui/card';
import { Sparkles } from 'lucide-react';

interface AISummaryProps {
  totalGoals: number;
  onTrackCount: number;
  criticalGoal?: string;
}

export function AIGoalsSummary({ totalGoals, onTrackCount, criticalGoal }: AISummaryProps) {
  return (
    <Card className="overflow-hidden border-none bg-linear-to-br from-indigo-500/10 via-purple-500/10 to-pink-500/10 backdrop-blur-sm">
      <CardContent className="p-6">
        <div className="flex items-start gap-4">
          <div className="p-2 bg-indigo-500 rounded-lg shrink-0">
            <Sparkles className="h-5 w-5 text-white" />
          </div>
          <div className="space-y-1">
            <h3 className="font-semibold text-lg flex items-center gap-2">AI Insight</h3>
            <p className="text-muted-foreground leading-relaxed">
              You have <span className="text-foreground font-medium">{totalGoals} active goals</span>. You're on track
              for <span className="text-emerald-500 font-medium">{onTrackCount} of them</span>.
              {criticalGoal && (
                <>
                  {' '}
                  Your{' '}
                  <span className="text-rose-500 font-medium underline underline-offset-4 decoration-rose-500/30">
                    {criticalGoal}
                  </span>{' '}
                  goal needs attention — at the current savings rate, you'll miss the deadline by 4 months.
                </>
              )}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
