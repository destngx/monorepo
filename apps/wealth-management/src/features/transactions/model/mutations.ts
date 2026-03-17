/**
 * Transactions Feature - Mutation Layer
 * 
 * Handles mutations (create, update, delete) for transactions.
 */

import { Transaction } from './types';

export async function createTransaction(data: Omit<Transaction, 'id' | 'accountBalance' | 'clearedBalance' | 'runningBalance' | 'cleared'>): Promise<Transaction> {
  const response = await fetch('/api/transactions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to create transaction');
  }
  return response.json();
}

export async function updateTransaction(id: string, data: Partial<Transaction>): Promise<Transaction> {
  const response = await fetch(`/api/transactions/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to update transaction');
  }
  return response.json();
}

export async function deleteTransaction(id: string): Promise<void> {
  const response = await fetch(`/api/transactions/${id}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete transaction');
  }
}
