/**
 * Goals Feature - Mutation Layer
 */

import { Goal } from './types';
import { GoalError } from '../../../utils/errors';

export async function createGoal(goal: Omit<Goal, 'id'>): Promise<Goal> {
  const response = await fetch('/api/goals', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(goal),
  });
  if (!response.ok) {
    throw new GoalError('Failed to create goal', {
      context: { endpoint: '/api/goals', statusCode: response.status },
      userMessage: 'Unable to create goal. Please try again.',
    });
  }
  return response.json();
}

export async function updateGoal(id: string, updates: Partial<Goal>): Promise<Goal> {
  const response = await fetch(`/api/goals/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  });
  if (!response.ok) {
    throw new GoalError('Failed to update goal', {
      context: { endpoint: `/api/goals/${id}`, statusCode: response.status },
      userMessage: 'Unable to update goal. Please try again.',
    });
  }
  return response.json();
}

export async function deleteGoal(id: string): Promise<void> {
  const response = await fetch(`/api/goals/${id}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete goal');
  }
}

export async function addContribution(goalId: string, amount: number, note?: string): Promise<Goal> {
  const response = await fetch(`/api/goals/${goalId}/contributions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ amount, note }),
  });
  if (!response.ok) {
    throw new Error('Failed to add contribution');
  }
  return response.json();
}
