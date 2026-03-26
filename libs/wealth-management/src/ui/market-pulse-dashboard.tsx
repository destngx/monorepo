'use client';

import { useState, useEffect } from 'react';
import useSWR from 'swr';
import { 
  AlertCircle, 
  RefreshCcw, 
  Globe, 
  Activity 
} from 'lucide-react';
import { Button } from './button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';

import { GLASS_CARD } from './market-pulse/constants';
import { MarketSection } from './market-pulse/MarketSection';

const fetcher = (url: string) => fetch(url).then((res) => res.json());

export function MarketPulseDashboard() {
  const [mounted, setMounted] = useState(false);
  const [timeframe, setTimeframe] = useState<string>('1h');
  const [autoRefresh, setAutoRefresh] = useState<string>('off');
  const [isRefreshing, setIsRefreshing] = useState(false);

  useEffect(() => setMounted(true), []);

  const refreshInterval = autoRefresh === 'off' ? 0 : parseInt(autoRefresh) * 1000;
  const { data, error, isLoading, mutate } = useSWR(`/api/market-pulse?timeframe=${timeframe}`, fetcher, {
    refreshInterval,
  });

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await mutate(fetch(`/api/market-pulse?timeframe=${timeframe}&force=true`).then((res) => res.json()));
    } finally {
      setIsRefreshing(false);
    }
  };

  if (error)
    return (
      <div className="p-8 text-center text-red-500 bg-red-500/10 rounded-xl border border-red-500/20">
        <AlertCircle className="w-10 h-10 mx-auto mb-4 opacity-50" />
        <h3 className="text-lg font-bold">Failed to load market pulse</h3>
        <p className="text-sm opacity-80">Check your connection or try again later.</p>
        <Button
          onClick={() => mutate()}
          variant="outline"
          className="mt-4 border-red-500/50 text-red-500 hover:bg-red-500/10"
        >
          Retry Load
        </Button>
      </div>
    );

  if (!mounted) return <div className="min-h-[600px] w-full animate-pulse bg-zinc-100/10 rounded-3xl" />;

  return (
    <div className="space-y-6 text-zinc-900 dark:text-zinc-100">
      <div className={`flex flex-col md:flex-row items-start md:items-center justify-between gap-4 p-4 rounded-2xl ${GLASS_CARD}`}>
        <div className="flex flex-wrap items-center gap-6">
          <div className="flex items-center gap-3">
            <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500">Market Timeframe</span>
            <Select value={timeframe} onValueChange={setTimeframe}>
              <SelectTrigger className="w-[100px] h-8 bg-zinc-100/50 dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800 text-xs font-mono">
                <SelectValue placeholder="Select" />
              </SelectTrigger>
              <SelectContent className="bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800">
                <SelectItem value="1h">1H CANDLE</SelectItem>
                <SelectItem value="4h">4H CANDLE</SelectItem>
                <SelectItem value="1d">1D SESSION</SelectItem>
                <SelectItem value="1w">1W TREND</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-[10px] font-bold uppercase tracking-[0.2em] text-zinc-500">Data Stream</span>
            <Select value={autoRefresh} onValueChange={setAutoRefresh}>
              <SelectTrigger className="w-[130px] h-8 bg-zinc-100/50 dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800 text-xs font-mono">
                <SelectValue placeholder="Select" />
              </SelectTrigger>
              <SelectContent className="bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800">
                <SelectItem value="off">REAL-TIME OFF</SelectItem>
                <SelectItem value="60">SYNC 1 MIN</SelectItem>
                <SelectItem value="300">SYNC 5 MIN</SelectItem>
                <SelectItem value="900">SYNC 15 MIN</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="flex items-center gap-6 w-full md:w-auto justify-between md:justify-end">
          <div className="text-[10px] text-zinc-400 font-mono flex items-center gap-2">
            <Activity className="w-3 h-3 text-emerald-500" />
            {data?.lastUpdated ? (
              <span className="opacity-80">CONNECTED: {new Date(data.lastUpdated).toLocaleTimeString()}</span>
            ) : isLoading ? (
              'SYNCHRONIZING...'
            ) : (
              'STANDBY'
            )}
          </div>
          <Button
            onClick={handleRefresh}
            disabled={isLoading || isRefreshing}
            variant="ghost"
            size="sm"
            className="h-8 w-8 p-0 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-all rounded-full"
          >
            <RefreshCcw className={`w-3.5 h-3.5 ${isLoading || isRefreshing ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </div>

      <Tabs defaultValue="vn" className="w-full">
        <div className="flex items-center justify-between mb-6">
          <TabsList className="bg-zinc-100 dark:bg-zinc-900/50 p-1 rounded-xl border border-zinc-200 dark:border-zinc-800/50">
            <TabsTrigger value="vn" className="px-6 py-2 rounded-lg data-[state=active]:bg-emerald-500/10 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-400 gap-2 transition-all font-bold">
              <Globe className="w-4 h-4" /> VIETNAM MARKET
            </TabsTrigger>
            <TabsTrigger value="us" className="px-6 py-2 rounded-lg data-[state=active]:bg-indigo-500/10 data-[state=active]:text-indigo-600 dark:data-[state=active]:text-indigo-400 gap-2 transition-all font-bold">
              <Globe className="w-4 h-4" /> US MARKETS
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="us" className="mt-0">
          <MarketSection market="US" marketData={data?.us} timeframe={timeframe} />
        </TabsContent>

        <TabsContent value="vn" className="mt-0">
          <MarketSection market="VN" marketData={data?.vn} timeframe={timeframe} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
