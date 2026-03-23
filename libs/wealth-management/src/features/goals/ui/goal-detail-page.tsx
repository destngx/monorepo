'use client';

import { use, useState, useEffect } from 'react';
import { Goal, GoalProjection } from '../model/types';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import {
  ArrowLeft,
  Plus,
  Calendar,
  Target,
  ShieldCheck,
  History as HistoryIcon,
  Sparkles,
  TrendingUp,
} from 'lucide-react';
import Link from 'next/link';
import { Badge } from '@/components/ui/badge';
import { MaskedBalance } from '@/components/ui/masked-balance';
import { GoalDetailChart } from './goal-detail-chart';
import { AIGoalInsights } from './ai-insights-panel';
import { cn } from '@wealth-management/utils';
import { formatVND } from '@wealth-management/utils';
import { AppError, isAppError } from '../../../utils/errors';

// In a real app, this would be fetched from an API
const MOCK_GOAL_DETAIL: Goal = {
  id: '1',
  name: 'Emergency Fund',
  type: 'SAVINGS_TARGET',
  emoji: '🏦',
  targetAmount: 100000000,
  currentAmount: 45000000,
  deadline: 'Dec 2026',
  status: 'ON_TRACK',
  linkedAccountId: 'acc_1',
  contributionType: 'AI_MANAGED',
  streakCount: 12,
  milestones: [
    { percentage: 25, label: 'Started', achievedAt: '2025-01-01' },
    { percentage: 50, label: 'Halfway' },
    { percentage: 75, label: 'Almost there' },
    { percentage: 100, label: 'Freedom!' },
  ],
  history: [
    { id: 'h1', date: '2026-03-01', amount: 2000000, sourceAccount: 'Checking', note: 'Monthly sweep' },
    { id: 'h2', date: '2026-02-15', amount: 500000, sourceAccount: 'Cash', note: 'Bonus' },
    { id: 'h3', date: '2026-02-01', amount: 2000000, sourceAccount: 'Checking', note: 'Monthly sweep' },
  ],
};

const MOCK_PROJECTION: GoalProjection = {
  currentPace: { completionDate: 'Dec 2026', monthlyContribution: 2500000 },
  requiredPace: { completionDate: 'Oct 2026', monthlyContribution: 3200000 },
  scenarios: [
    { label: 'Current pace', monthlyContribution: 2500000, completionDate: 'Dec 2026' },
    { label: '+₫700K/month', monthlyContribution: 3200000, completionDate: 'Oct 2026' },
    { label: '+₫1.5M/month', monthlyContribution: 4000000, completionDate: 'Aug 2026' },
  ],
};

const CHART_DATA = [
  { date: 'Jan', actual: 35000000, target: 100000000 },
  { date: 'Feb', actual: 38000000, target: 100000000 },
  { date: 'Mar', actual: 45000000, target: 100000000, projected: 45000000 },
  { date: 'Apr', actual: 45000000, projected: 48000000, target: 100000000 },
  { date: 'May', actual: 45000000, projected: 52000000, target: 100000000 },
  { date: 'Jun', actual: 45000000, projected: 56000000, target: 100000000 },
];

export default function GoalDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const [goal, setGoal] = useState<Goal | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // In a real app, we'd fetch specific ID.
    // Here we'll fetch all and find by ID (id is goal-index)
    fetch('/api/goals')
      .then((res) => res.json())
      .then((data: Goal[]) => {
        const found = data.find((g) => g.id === resolvedParams.id);
        if (found) setGoal(found);
        setLoading(false);
      })
      .catch((err) => {
        if (isAppError(err)) {
          console.error('Failed to load goal:', err.message);
        } else {
          const error = new AppError(err instanceof Error ? err.message : 'Failed to load goal');
          console.error('Failed to load goal:', error.message);
        }
        setLoading(false);
      });
  }, [resolvedParams.id]);

  if (loading) return <div>Loading...</div>;
  if (!goal) return <div>Goal not found</div>;

  const progress = (goal.currentAmount / goal.targetAmount) * 100;

  return (
    <div className="max-w-4xl mx-auto space-y-8 pb-20">
      {/* Header */}
      <div className="flex items-center justify-between">
        <Link href="/accounts/goals">
          <Button variant="ghost" size="sm" className="gap-2 text-muted-foreground hover:text-foreground">
            <ArrowLeft className="h-4 w-4" />
            Back to Goals
          </Button>
        </Link>
        <div className="flex items-center gap-2">
          <Badge
            variant="outline"
            className="bg-indigo-500/5 text-indigo-500 border-indigo-500/20 px-3 py-1 font-bold uppercase tracking-tighter text-[10px]"
          >
            AI Managed
          </Badge>
        </div>
      </div>

      {/* Hero Section */}
      <div className="relative overflow-hidden rounded-3xl bg-linear-to-br from-indigo-600 to-purple-700 text-white p-8 md:p-12 shadow-2xl shadow-indigo-500/20">
        <div className="absolute top-0 right-0 p-8 opacity-10">
          <Target size={200} />
        </div>

        <div className="relative flex flex-col items-center text-center space-y-6">
          <div className="text-6xl mb-2">{goal.emoji}</div>
          <h1 className="text-4xl md:text-5xl font-black tracking-tight">{goal.name}</h1>

          <div className="w-full max-w-md space-y-4">
            <div className="relative h-4 w-full bg-white/20 rounded-full overflow-hidden">
              <div
                className="absolute inset-y-0 left-0 bg-white rounded-full transition-all duration-1000 ease-in-out"
                style={{ width: `${progress}%` }}
              />
            </div>
            <div className="flex justify-between text-sm font-bold uppercase tracking-widest text-white/70">
              <span>{progress.toFixed(0)}% Complete</span>
              <span>Target: {formatVND(goal.targetAmount)}</span>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-8 w-full pt-4">
            <div className="text-center">
              <p className="text-xs uppercase tracking-widest text-white/50 mb-1 font-bold">Saved</p>
              <p className="text-xl font-bold">
                <MaskedBalance amount={goal.currentAmount} />
              </p>
            </div>
            <div className="text-center">
              <p className="text-xs uppercase tracking-widest text-white/50 mb-1 font-bold">Remaining</p>
              <p className="text-xl font-bold">
                <MaskedBalance amount={goal.targetAmount - goal.currentAmount} />
              </p>
            </div>
            <div className="text-center">
              <p className="text-xs uppercase tracking-widest text-white/50 mb-1 font-bold">Deadline</p>
              <p className="text-xl font-bold">{goal.deadline}</p>
            </div>
          </div>

          <Button className="mt-8 bg-white text-indigo-600 hover:bg-white/90 rounded-full px-8 py-6 h-auto font-black text-lg uppercase tracking-wider gap-3 shadow-lg">
            <Plus className="h-6 w-6" />
            Add Contribution
          </Button>
        </div>
      </div>

      {/* Tabs / Content Sections */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Left Column: Progress & History */}
        <div className="lg:col-span-2 space-y-8">
          <Card className="border-none shadow-sm overflow-hidden">
            <CardContent className="p-6">
              <div className="flex items-center justify-between mb-6">
                <h3 className="font-bold flex items-center gap-2 uppercase tracking-tight text-sm">
                  <TrendingUp className="h-4 w-4 text-indigo-500" />
                  Progress Path
                </h3>
                <div className="flex gap-1">
                  {['W', 'M', 'Y', 'ALL'].map((t) => (
                    <Button
                      key={t}
                      variant="ghost"
                      size="sm"
                      className={cn('h-7 px-2 text-[10px] font-bold', t === 'M' && 'bg-muted text-foreground')}
                    >
                      {t}
                    </Button>
                  ))}
                </div>
              </div>
              <GoalDetailChart data={CHART_DATA} />
            </CardContent>
          </Card>

          <Card className="border-none shadow-sm overflow-hidden">
            <CardContent className="p-0">
              <div className="p-6 pb-2">
                <h3 className="font-bold flex items-center gap-2 uppercase tracking-tight text-sm">
                  <HistoryIcon className="h-4 w-4 text-indigo-500" />
                  Contribution History
                </h3>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="bg-muted/50 text-muted-foreground text-[10px] uppercase font-bold text-left">
                      <th className="px-6 py-3">Date</th>
                      <th className="px-6 py-3">Source</th>
                      <th className="px-6 py-3 text-right">Amount</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-muted/50">
                    {goal.history.map((h) => (
                      <tr key={h.id} className="hover:bg-muted/30 transition-colors">
                        <td className="px-6 py-4 font-medium">{h.date}</td>
                        <td className="px-6 py-4 text-muted-foreground text-xs">{h.sourceAccount}</td>
                        <td className="px-6 py-4 text-right font-bold text-indigo-500">{formatVND(h.amount)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Right Column: AI Coaching */}
        <div className="space-y-6">
          <div className="flex items-center gap-2 mb-2 p-1">
            <Sparkles className="h-5 w-5 text-indigo-500 fill-indigo-500/20" />
            <h3 className="font-black uppercase tracking-tighter text-lg italic">AI Advisor</h3>
          </div>
          <AIGoalInsights projection={MOCK_PROJECTION} />

          <Card className="border-dashed border-2 bg-indigo-500/5 hover:bg-indigo-500/10 transition-colors cursor-pointer group">
            <CardContent className="p-6 space-y-3">
              <div className="flex items-center gap-2 text-indigo-500">
                <ShieldCheck className="h-5 w-5" />
                <span className="font-bold text-sm uppercase tracking-tight">Milestone Streak</span>
              </div>
              <p className="text-sm text-balance leading-relaxed">
                You've maintained a <span className="font-bold">12-week streak</span>! You're just{' '}
                <span className="font-bold">₫5.0M</span> away from your next 75% milestone.
              </p>
              <div className="flex gap-2 pt-2 overflow-x-auto pb-2 noscrollbar">
                {goal.milestones.map((m, i) => (
                  <div key={i} className={cn('h-2 w-full rounded-full', i < 2 ? 'bg-indigo-500' : 'bg-muted')} />
                ))}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
