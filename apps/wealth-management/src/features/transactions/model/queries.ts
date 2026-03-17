/**
 * Transactions Feature - Query Layer
 * 
 * Handles fetching transaction data from the API/persistence layer.
 */

import { Transaction } from './types';

export async function getTransactions(): Promise<Transaction[]> {
  const response = await fetch('/api/transactions');
  if (!response.ok) {
    throw new Error('Failed to fetch transactions');
  }
  return response.json();
}

export async function getTransactionsByAccount(accountName: string): Promise<Transaction[]> {
  const transactions = await getTransactions();
  return transactions.filter(t => t.accountName === accountName);
}

export async function getTransactionsByCategory(category: string): Promise<Transaction[]> {
  const transactions = await getTransactions();
  return transactions.filter(t => t.category === category);
}

export async function getTransactionsByDateRange(startDate: Date, endDate: Date): Promise<Transaction[]> {
  const transactions = await getTransactions();
  return transactions.filter(t => {
    const txDate = new Date(t.date);
    return txDate >= startDate && txDate <= endDate;
  });
}
