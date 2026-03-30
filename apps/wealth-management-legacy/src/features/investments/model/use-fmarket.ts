'use client';

import { useState, useEffect } from 'react';
import { fmarketApi } from '@/shared/api/fmarket';

export interface FmarketFund {
  id: number;
  name: string;
  code: string;
  nav: number;
  navTo12Months: number;
  annualizedReturn36Months: number;
  fundAssetType: string;
  issuerName: string;
  issuerShortName: string;
}

export interface BankRate {
  id: number;
  bankName: string;
  bankShortName: string;
  interestRate: number;
  period: number;
  type: string;
}

export type FmarketRange = '1M' | '6M' | 'YTD' | '1Y' | '2Y' | '3Y' | '5Y' | 'ALL';

export function useFmarket() {
  const [stockFunds, setStockFunds] = useState<any[]>([]);
  const [bondFunds, setBondFunds] = useState<any[]>([]);
  const [balancedFunds, setBalancedFunds] = useState<any[]>([]);
  const [mmfFunds, setMmfFunds] = useState<any[]>([]);
  const [issuers, setIssuers] = useState<any[]>([]);
  const [bankRates, setBankRates] = useState<any[]>([]);
  const [goldHistory, setGoldHistory] = useState<any[]>([]);
  const [usdHistory, setUsdHistory] = useState<any[]>([]);
  const [goldProducts, setGoldProducts] = useState<any[]>([]);
  const [goldExtra, setGoldExtra] = useState<any>(null);
  const [bankData, setBankData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [goldLoading, setGoldLoading] = useState(false);
  const [usdLoading, setUsdLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [goldRange, setGoldRange] = useState<FmarketRange>('1Y');
  const [usdRange, setUsdRange] = useState<FmarketRange>('1Y');

  const getDatesForRange = (range: FmarketRange) => {
    const now = new Date();
    const from = new Date();

    switch (range) {
      case '1M':
        from.setMonth(now.getMonth() - 1);
        break;
      case 'YTD':
        from.setMonth(0, 1);
        break;
      case '6M':
        from.setMonth(now.getMonth() - 6);
        break;
      case '1Y':
        from.setFullYear(now.getFullYear() - 1);
        break;
      case '2Y':
        from.setFullYear(now.getFullYear() - 2);
        break;
      case '3Y':
        from.setFullYear(now.getFullYear() - 3);
        break;
      case '5Y':
        from.setFullYear(now.getFullYear() - 5);
        break;
      case 'ALL':
        from.setFullYear(now.getFullYear() - 10);
        break;
    }

    const formatDate = (date: Date) => {
      const y = date.getFullYear();
      const m = String(date.getMonth() + 1).padStart(2, '0');
      const d = String(date.getDate()).padStart(2, '0');
      return `${y}${m}${d}`;
    };

    return {
      fromDate: formatDate(from),
      toDate: formatDate(now),
    };
  };

  useEffect(() => {
    const fetchData = async () => {
      try {
        const { fromDate, toDate } = getDatesForRange('1Y');

        const [
          stockData,
          bondData,
          balancedData,
          mmfData,
          issuersData,
          ratesData,
          goldHistoryData,
          usdData,
          goldProductsData,
        ] = await Promise.all([
          fmarketApi.getProductsFilterNav(1, 1000, ['STOCK']),
          fmarketApi.getProductsFilterNav(1, 1000, ['BOND']),
          fmarketApi.getProductsFilterNav(1, 1000, ['BALANCED']),
          fmarketApi.getProductsFilterNav(1, 1000, [], true), // isMMFFund: true
          fmarketApi.getIssuers(),
          fmarketApi.getBankInterestRates(),
          fmarketApi.getGoldPriceHistory(fromDate, toDate),
          fmarketApi.getUsdRateHistory(fromDate, toDate),
          fmarketApi.getGoldProducts(),
        ]);

        setStockFunds(stockData?.data?.rows || stockData?.data?.list || []);
        setBondFunds(bondData?.data?.rows || bondData?.data?.list || []);
        setBalancedFunds(balancedData?.data?.rows || balancedData?.data?.list || []);
        setMmfFunds(mmfData?.data?.rows || mmfData?.data?.list || []);
        setIssuers(issuersData?.data?.rows || issuersData?.data || []);
        setGoldHistory(goldHistoryData?.data || []);
        setUsdHistory(usdData?.data || []);
        setGoldProducts(goldProductsData?.data?.rows || []);
        setGoldExtra(goldHistoryData?.extra || null);

        const rates = ratesData?.data?.bankList;
        setBankRates(Array.isArray(rates) ? rates : []);
        setBankData(ratesData?.data || null);
      } catch (err) {
        setError(err instanceof Error ? err : new Error('Failed to fetch Fmarket data'));
      } finally {
        setLoading(false);
      }
    };

    void fetchData();
  }, []);

  useEffect(() => {
    const fetchGold = async () => {
      setGoldLoading(true);
      try {
        const { fromDate, toDate } = getDatesForRange(goldRange);
        const res = await fmarketApi.getGoldPriceHistory(fromDate, toDate);
        setGoldHistory(res?.data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setGoldLoading(false);
      }
    };
    if (!loading) void fetchGold();
  }, [goldRange, loading]);

  useEffect(() => {
    const fetchUsd = async () => {
      setUsdLoading(true);
      try {
        const { fromDate, toDate } = getDatesForRange(usdRange);
        const res = await fmarketApi.getUsdRateHistory(fromDate, toDate);
        setUsdHistory(res?.data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setUsdLoading(false);
      }
    };
    if (!loading) void fetchUsd();
  }, [usdRange, loading]);

  const [navHistory, setNavHistory] = useState<any[]>([]);
  const [detailsLoading, setDetailsLoading] = useState(false);
  const [selectedTicker, setSelectedTicker] = useState<string | null>(null);
  const [tickerDetails, setTickerDetails] = useState<any>(null);
  const [navRange, setNavRange] = useState<FmarketRange>('1Y');

  const navRangeMap: Record<FmarketRange, string> = {
    '1M': 'navTo1Month',
    '6M': 'navTo6Months',
    YTD: 'navToLastYear',
    '1Y': 'navTo12Months',
    '2Y': 'navTo24Months',
    '3Y': 'navTo36Months',
    '5Y': 'navTo60Months',
    ALL: 'navToBeginning',
  };

  useEffect(() => {
    const fetchHistory = async () => {
      if (!selectedTicker || !tickerDetails?.id) return;

      try {
        const history = await fmarketApi.getNavHistory(tickerDetails.id, navRangeMap[navRange]);
        if (history?.status === 200) {
          setNavHistory(history.data || []);
        }
      } catch (err) {
        console.error('Error fetching nav history:', err);
      }
    };
    if (tickerDetails?.id) void fetchHistory();
  }, [navRange, tickerDetails?.id]);

  useEffect(() => {
    const fetchDetails = async () => {
      if (!selectedTicker) {
        setTickerDetails(null);
        setNavHistory([]);
        return;
      }

      setDetailsLoading(true);
      try {
        const details = await fmarketApi.getProductDetails(selectedTicker);
        if (details?.status === 200 && details?.data) {
          setTickerDetails(details.data);

          if (details.data.id) {
            const history = await fmarketApi.getNavHistory(details.data.id);
            if (history?.status === 200) {
              setNavHistory(history.data || []);
            }
          }
        } else {
          console.error('Failed to fetch details:', details?.message);
          setTickerDetails(null);
        }
      } catch (err) {
        console.error('Error fetching ticker details:', err);
        setTickerDetails(null);
      } finally {
        setDetailsLoading(false);
      }
    };
    void fetchDetails();
  }, [selectedTicker]);

  return {
    stockFunds,
    bondFunds,
    balancedFunds,
    mmfFunds,
    issuers,
    bankRates,
    bankData,
    goldHistory,
    usdHistory,
    goldProducts,
    goldExtra,
    loading,
    goldLoading,
    usdLoading,
    detailsLoading,
    error,
    goldRange,
    setGoldRange,
    usdRange,
    setUsdRange,
    navRange,
    setNavRange,
    selectedTicker,
    setSelectedTicker,
    tickerDetails,
    navHistory,
  };
}
