/**
 * Loans Feature - React Hooks
 */

'use client';

import { useEffect, useState } from 'react';
import { Loan } from './types';
import { AppError, isAppError, getErrorMessage } from '../../../utils/errors';
import * as queries from './queries';

export function useLoans() {
  const [loans, setLoans] = useState<Loan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    queries
      .getLoans()
      .then(setLoans)
      .catch((err: unknown) => {
        const appError = isAppError(err) ? err : new AppError(getErrorMessage(err));
        setError(appError);
      })
      .finally(() => setLoading(false));
  }, []);

  return { loans, loading, error };
}

export function useLoan(name: string) {
  const [loan, setLoan] = useState<Loan | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    queries
      .getLoanByName(name)
      .then(setLoan)
      .catch((err: unknown) => {
        const appError = isAppError(err) ? err : new AppError(getErrorMessage(err));
        setError(appError);
      })
      .finally(() => setLoading(false));
  }, [name]);

  return { loan, loading, error };
}
