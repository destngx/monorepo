/**
 * Goals Feature - Query Layer
 */

import { Goal, GoalProjection } from './types';

export async function getGoals(): Promise<Goal[]> {
  const response = await fetch('/api/goals');
  if (!response.ok) {
    throw new Error('Failed to fetch goals');
  }
  return response.json();
}

export async function getGoalById(id: string): Promise<Goal | null> {
  const goals = await getGoals();
  return goals.find(goal => goal.id === id) || null;
}

export async function getGoalProjection(id: string): Promise<GoalProjection | null> {
  const goal = await getGoalById(id);
  if (!goal) return null;

  const daysRemaining = Math.max(
    0,
    (new Date(goal.deadline).getTime() - Date.now()) / (1000 * 60 * 60 * 24)
  );
  const amountRemaining = Math.max(0, goal.targetAmount - goal.currentAmount);
  const monthlyRequired = daysRemaining > 0 ? (amountRemaining / daysRemaining) * 30 : 0;

  return {
    currentPace: {
      completionDate: goal.deadline,
      monthlyContribution: goal.history.length > 0 
        ? goal.currentAmount / (goal.history.length / 12)
        : 0,
    },
    requiredPace: {
      completionDate: goal.deadline,
      monthlyContribution: monthlyRequired,
    },
    scenarios: [
      { label: 'Current Pace', monthlyContribution: 0, completionDate: goal.deadline },
      { label: '+50%', monthlyContribution: monthlyRequired * 1.5, completionDate: goal.deadline },
      { label: '+100%', monthlyContribution: monthlyRequired * 2, completionDate: goal.deadline },
    ],
  };
}
