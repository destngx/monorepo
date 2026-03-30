'use client';

/**
 * Budget Feature - Custom Hooks
 */

import { useState, useEffect } from 'react';
import { BudgetItem } from './types';
import { getBudgetItems } from './queries';

export function useBudgetItems() {
  const [items, setItems] = useState<BudgetItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchItems = async () => {
      try {
        setLoading(true);
        const data = await getBudgetItems();
        setItems(data);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        setLoading(false);
      }
    };

    void fetchItems();
  }, []);

  return { items, loading, error };
}
