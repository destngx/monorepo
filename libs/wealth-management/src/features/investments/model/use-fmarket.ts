import { useState } from 'react';
import { GoldExtra, UsdHistoryRaw, FmarketFund, GoldHistoryRaw } from './types';

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
    stockFunds: [] as FmarketFund[],
    bondFunds: [] as FmarketFund[],
    balancedFunds: [] as FmarketFund[],
    mmfFunds: [] as FmarketFund[],
    bankRates: [] as any[],
    goldHistory: [] as GoldHistoryRaw[],
    usdHistory: [] as UsdHistoryRaw[],
    goldProducts: [] as any[],
    goldExtra: null as GoldExtra | null,
    bankData: [] as any[],
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
    tickerDetails: null as any,
    navHistory: [] as any[],
    detailsLoading: false,
    navRange: '1m' as const,
    setNavRange: (range: string) => {},
  };
}
