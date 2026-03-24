'use client';

import { useState, useEffect, useMemo } from 'react';
import { AIGoalsSummary } from './ai-summary-card';
import { GoalCard } from './goal-card';
import { Goal } from '../model/types';
import { Button } from '@/components/ui/button';
import Link from 'next/link';
import { Plus, SlidersHorizontal, TrendingUp } from 'lucide-react';
import { useAIContext } from '@/components/chat/ai-context-provider';
import { useApiErrorHandler } from '../../../hooks/use-api-error-handler';

// Mock data for initial implementation
const MOCK_GOALS: Goal[] = [
  {
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
    ],
    history: [],
  },
  {
    id: '2',
    name: 'New MacBook Pro',
    type: 'PURCHASE_GOAL',
    emoji: '💻',
    targetAmount: 65000000,
    currentAmount: 20000000,
    deadline: 'Aug 2026',
    status: 'AT_RISK',
    linkedAccountId: 'acc_2',
    contributionType: 'MANUAL',
    streakCount: 4,
    milestones: [],
    history: [],
  },
  {
    id: '3',
    name: 'Europe 2027',
    type: 'SAVINGS_TARGET',
    emoji: '✈️',
    targetAmount: 150000000,
    currentAmount: 85000000,
    deadline: 'Jun 2027',
    status: 'ON_TRACK',
    linkedAccountId: 'acc_1',
    contributionType: 'AUTOMATIC',
    streakCount: 8,
    milestones: [],
    history: [],
  },
];

export default function GoalsPage() {
  const [goals, setGoals] = useState<Goal[]>([]);
  const [loading, setLoading] = useState(true);
  const { setPageData, addInsight } = useAIContext();
  const { withErrorHandling } = useApiErrorHandler();

  useEffect(() => {
    withErrorHandling(async () => {
      const response = await fetch('/api/goals');
      if (!response.ok) throw new Error(`Failed: ${response.statusText}`);
      return response.json() as Promise<Goal[]>;
    }, 'Failed to load goals')
      .then((data: Goal[] | null) => {
        if (!data) {
          setLoading(false);
          return;
        }

        const goalsData = data as Goal[];
        setGoals(goalsData);
        setLoading(false);

        setPageData({
          goals: goalsData,
          totalTarget: goalsData.reduce((acc: number, g: Goal) => acc + g.targetAmount, 0),
          totalSaved: goalsData.reduce((acc: number, g: Goal) => acc + g.currentAmount, 0),
        });

        if (goalsData.some((g: Goal) => g.status === 'AT_RISK')) {
          addInsight({
            type: 'alert',
            title: 'Goal at Risk',
            content:
              "Your MacBook Pro goal is currently at risk due to a slow-down in contributions. If you continue at this pace, you'll reach it 3 months later than planned.",
            suggestedQuestions: [
              'How can I fix my MacBook goal?',
              'Show me my goal timeline',
              'Can I still reach it by August?',
            ],
          });
        }
      })
      .catch((error: unknown) => {
        console.error('Failed to fetch goals:', error);
        setLoading(false);
      });
  }, [setPageData, addInsight, withErrorHandling]);

  if (loading) {
    return (
      <div className="flex items-center justify-center p-20">
        <p className="text-muted-foreground animate-pulse">Scanning targets...</p>
      </div>
    );
  }

  return (
    <div className="space-y-8 max-w-7xl mx-auto pb-20">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight mb-2">My Goals</h1>
          <p className="text-muted-foreground">
            Transforming your abstract financial intentions into concrete milestones.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" className="gap-2">
            <SlidersHorizontal className="h-4 w-4" />
            Sort
          </Button>
          <Link href="/accounts/goals/new">
            <Button size="sm" className="gap-2 bg-indigo-600 hover:bg-indigo-700 text-white">
              <Plus className="h-4 w-4" />
              New Goal
            </Button>
          </Link>
        </div>
      </div>

      <AIGoalsSummary
        totalGoals={goals.length}
        onTrackCount={goals.filter((g) => g.status === 'ON_TRACK').length}
        criticalGoal={goals.find((g) => g.status === 'AT_RISK' || g.status === 'OFF_TRACK')?.name}
      />

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {goals.map((goal) => (
          <GoalCard key={goal.id} goal={goal} />
        ))}
      </div>
    </div>
  );
}
