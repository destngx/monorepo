'use client';

/**
 * Budget Feature - Custom Hooks
 */

import { useState, useEffect } from 'react';
import { BudgetItem } from './types';
import { getBudgetItems } from './queries';
import { AppError } from '../../../utils/errors';

export function useBudgetItems() {
  const [items, setItems] = useState<BudgetItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<AppError | null>(null);

  useEffect(() => {
    const fetchItems = async () => {
      try {
        setLoading(true);
        const data = await getBudgetItems();
        setItems(data);
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

    fetchItems();
  }, []);

  return { items, loading, error };
}
