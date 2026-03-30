/**
 * Accounts Feature - Mutation Layer
 *
 * Handles mutations (create, update, delete) for accounts.
 */

import { Account } from './types';

export async function createAccount(data: Partial<Account>): Promise<Account> {
  const response = await fetch('/api/accounts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to create account');
  }
  return response.json();
}

export async function updateAccount(name: string, data: Partial<Account>): Promise<Account> {
  const response = await fetch(`/api/accounts/${name}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
  if (!response.ok) {
    throw new Error('Failed to update account');
  }
  return response.json();
}

export async function deleteAccount(name: string): Promise<void> {
  const response = await fetch(`/api/accounts/${name}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to delete account');
  }
}
