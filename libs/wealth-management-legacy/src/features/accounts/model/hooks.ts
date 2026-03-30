'use client';

/**
 * Accounts Feature - Custom Hooks
 */

import { useState, useEffect } from 'react';
import { Account } from './types';
import { getAccounts } from './queries';
import { AppError } from '../../../utils/errors';

export function useAccounts() {
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    const fetchAccounts = async () => {
      try {
        setLoading(true);
        const data = await getAccounts();
        setAccounts(data);
      } catch (err) {
        if (err instanceof AppError) {
          setError(err);
        } else {
          const appError = new AppError(err instanceof Error ? err.message : 'Unknown error');
          setError(appError);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchAccounts();
  }, []);

  return { accounts, loading, error };
}

export function useAccountById(name: string | null) {
  const [account, setAccount] = useState<Account | null>(null);
  const [loading, setLoading] = useState(!!name);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    if (!name) {
      setAccount(null);
      return;
    }

    const fetchAccount = async () => {
      try {
        setLoading(true);
        const accounts = await getAccounts();
        const found = accounts.find((a) => a.name === name);
        setAccount(found || null);
      } catch (err) {
        if (err instanceof AppError) {
          setError(err);
        } else {
          const appError = new AppError(err instanceof Error ? err.message : 'Unknown error');
          setError(appError);
        }
      } finally {
        setLoading(false);
      }
    };

    fetchAccount();
  }, [name]);

  return { account, loading, error };
}
