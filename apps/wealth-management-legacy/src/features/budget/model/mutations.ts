/**
 * Budget Feature - Mutation Layer
 */

import { BudgetItem } from './types';

export async function updateBudgetItem(category: string, data: Partial<BudgetItem>): Promise<BudgetItem> {
  const response = await fetch(`/api/budget/${category}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to update budget item');
  }
  return response.json();
}

export async function createBudgetItem(data: Partial<BudgetItem>): Promise<BudgetItem> {
  const response = await fetch('/api/budget', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to create budget item');
  }
  return response.json();
}

export async function deleteBudgetItem(category: string): Promise<void> {
  const response = await fetch(`/api/budget/${category}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete budget item');
  }
}
