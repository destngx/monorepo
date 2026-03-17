'use client';

/**
 * Investments Feature - Custom Hooks
 */

import { useState, useEffect } from 'react';
import { getAssets, getAssetPrices } from './queries';
import type { Asset } from './queries';

export function useAssets() {
  const [assets, setAssets] = useState<Asset[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchAssets = async () => {
      try {
        setLoading(true);
        const data = await getAssets();
        setAssets(data);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        setLoading(false);
      }
    };

    fetchAssets();
  }, []);

  return { assets, loading, error };
}

export function useAssetPrices() {
  const [prices, setPrices] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const fetchPrices = async () => {
      try {
        setLoading(true);
        const data = await getAssetPrices();
        setPrices(data);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
      } finally {
        setLoading(false);
      }
    };

    fetchPrices();
  }, []);

  return { prices, loading, error };
}
