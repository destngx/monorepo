/**
 * Loans Feature - Query Layer
 */

import { Loan } from './types';

export async function getLoans(): Promise<Loan[]> {
  const response = await fetch('/api/loans');
  if (!response.ok) {
    throw new Error('Failed to fetch loans');
  }
  return response.json();
}

export async function getLoanByName(name: string): Promise<Loan | null> {
  const loans = await getLoans();
  return loans.find(loan => loan.name === name) || null;
}

export async function getTotalMonthlyDebt(): Promise<number> {
  const loans = await getLoans();
  return loans.reduce((sum, loan) => sum + loan.monthlyDebt, 0);
}

export async function getTotalMonthlyPaid(): Promise<number> {
  const loans = await getLoans();
  return loans.reduce((sum, loan) => sum + loan.monthlyPaid, 0);
}

export async function getTotalMonthlyRemaining(): Promise<number> {
  const loans = await getLoans();
  return loans.reduce((sum, loan) => sum + loan.monthlyRemaining, 0);
}
