/**
 * Loans Feature - Mutation Layer
 */

import { Loan } from './types';

export async function createLoan(loan: Loan): Promise<Loan> {
  const response = await fetch('/api/loans', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(loan),
  });
  if (!response.ok) {
    throw new Error('Failed to create loan');
  }
  return response.json();
}

export async function updateLoan(name: string, updates: Partial<Loan>): Promise<Loan> {
  const response = await fetch(`/api/loans/${name}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(updates),
  });
  if (!response.ok) {
    throw new Error('Failed to update loan');
  }
  return response.json();
}

export async function deleteLoan(name: string): Promise<void> {
  const response = await fetch(`/api/loans/${name}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete loan');
  }
}
