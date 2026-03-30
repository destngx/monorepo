/**
 * Budget Feature - Query Layer
 */

import { BudgetItem } from './types';

export async function getBudgetItems(): Promise<BudgetItem[]> {
  const response = await fetch('/api/budget');
  if (!response.ok) {
    throw new Error('Failed to fetch budget items');
  }
  return response.json();
}

export async function getBudgetByCategory(category: string): Promise<BudgetItem | null> {
  const items = await getBudgetItems();
  return items.find((item) => item.category === category) || null;
}

export async function getTotalBudgetSpent(): Promise<number> {
  const items = await getBudgetItems();
  return items.reduce((sum, item) => sum + (item.monthlySpent || 0), 0);
}
