/**
 * Accounts Feature - Query Layer
 * 
 * Handles fetching account data from the API/persistence layer.
 */

import { Account } from './types';

export async function getAccounts(): Promise<Account[]> {
  const response = await fetch('/api/accounts');
  if (!response.ok) {
    throw new Error('Failed to fetch accounts');
  }
  return response.json();
}

export async function getAccountById(name: string): Promise<Account | null> {
  const accounts = await getAccounts();
  return accounts.find(a => a.name === name) || null;
}

export async function getAccountsByType(type: string): Promise<Account[]> {
  const accounts = await getAccounts();
  return accounts.filter(a => a.type === type);
}
