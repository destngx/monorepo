'use client';

/**
 * Transactions Feature - Custom Hooks
 */

import { useState, useEffect } from 'react';
import { Transaction } from './types';
import { getTransactions } from './queries';

export function useTransactions() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchTransactions = async () => {
      try {
        setLoading(true);
        const data = await getTransactions();
        setTransactions(data);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        setLoading(false);
      }
    };

    void fetchTransactions();
  }, []);

  return { transactions, loading, error };
}

export function useTransactionsByAccount(accountName: string | null) {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(!!accountName);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    if (!accountName) {
      setTransactions([]);
      return;
    }

    const fetchTransactions = async () => {
      try {
        setLoading(true);
        const allTransactions = await getTransactions();
        const filtered = allTransactions.filter((t) => t.accountName === accountName);
        setTransactions(filtered);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        setLoading(false);
      }
    };

    void fetchTransactions();
  }, [accountName]);

  return { transactions, loading, error };
}
