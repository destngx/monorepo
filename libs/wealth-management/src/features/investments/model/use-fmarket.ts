import { useState } from 'react';

/**
 * Hook for FMarket data - stub implementation
 * TODO: Implement actual FMarket data fetching
 */
export function useFmarket() {
  const [loading, setLoading] = useState(false);
  const [goldLoading, setGoldLoading] = useState(false);
  const [usdLoading, setUsdLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedTicker, setSelectedTickerState] = useState<string | null>(null);

  const setSelectedTicker = (ticker: string | null) => {
    setSelectedTickerState(ticker);
  };

  return {
    stockFunds: [],
    bondFunds: [],
    balancedFunds: [],
    mmfFunds: [],
    bankRates: [],
    goldHistory: [],
    usdHistory: [],
    goldProducts: [],
    goldExtra: null,
    bankData: [],
    loading,
    goldLoading,
    usdLoading,
    error,
    goldRange: '1m' as const,
    setGoldRange: (range: string) => {},
    usdRange: '1m' as const,
    setUsdRange: (range: string) => {},
    selectedTicker,
    setSelectedTicker,
    tickerDetails: null,
    navHistory: [],
    detailsLoading: false,
    navRange: '1m' as const,
    setNavRange: (range: string) => {},
  };
}
