"use client";

import { useState, useEffect } from "react";
import useSWR from "swr";
import { useTheme } from "next-themes";
import { type MarketAsset } from "@wealth-management/services";
import { 
  BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Cell, ReferenceLine
} from "recharts";
import { 
  Card, CardContent, CardHeader, CardTitle 
} from "@/components/ui/card";
import { 
  TrendingUp, Activity, AlertCircle, Info, RefreshCcw, 
  ArrowUpRight, ArrowDownRight, Globe, Zap, Shield, Flame, Moon, Search,
  Compass, BarChart3, Binary, PieChart, Layers
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { 
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue 
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { 
  Tooltip as ShadcnTooltip, TooltipContent, TooltipProvider, TooltipTrigger 
} from "@/components/ui/tooltip";
import {
  Tabs, TabsContent, TabsList, TabsTrigger
} from "@/components/ui/tabs";
import { AIDataInsight } from "@/components/dashboard/ai-data-insight";

const GLASS_CARD = "bg-white/60 dark:bg-zinc-900/40 backdrop-blur-md border-zinc-200 dark:border-zinc-800/50 shadow-lg dark:shadow-2xl relative overflow-hidden";
const TERMINAL_FONT = "font-mono tracking-tight";

const fetcher = (url: string) => fetch(url).then(res => res.json());

export function MarketPulseDashboard() {
  const { theme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const [timeframe, setTimeframe] = useState<string>("1h");
  const [autoRefresh, setAutoRefresh] = useState<string>("off");
  const [isRefreshing, setIsRefreshing] = useState(false);

  // Avoid hydration mismatch for theme-dependent components
  useEffect(() => setMounted(true), []);
  
  const refreshInterval = autoRefresh === "off" ? 0 : parseInt(autoRefresh) * 1000;
  const { data, error, isLoading, mutate } = useSWR(
    `/api/market-pulse?timeframe=${timeframe}`, 
    fetcher, 
    { refreshInterval }
  );

  const handleRefresh = async () => {
    setIsRefreshing(true);
    try {
      await mutate(fetch(`/api/market-pulse?timeframe=${timeframe}&force=true`).then(res => res.json()));
    } finally {
      setIsRefreshing(false);
    }
  };

  if (error) return (
    <div className="p-8 text-center text-red-500 bg-red-500/10 rounded-xl border border-red-500/20">
      <AlertCircle className="w-10 h-10 mx-auto mb-4 opacity-50" />
      <h3 className="text-lg font-bold">Failed to load market pulse</h3>
      <p className="text-sm opacity-80">Check your connection or try again later.</p>
      <Button onClick={() => mutate()} variant="outline" className="mt-4 border-red-500/50 text-red-500 hover:bg-red-500/10">
        Retry Load
      </Button>
    </div>
  );

  if (!mounted) return <div className="min-h-[600px] w-full animate-pulse bg-zinc-100/10 rounded-3xl" />;

  return (
    <div className="space-y-6 text-zinc-900 dark:text-zinc-100">
      {/* Premium Controls Bar */}
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
            ) : (
              isLoading ? "SYNCHRONIZING..." : "STANDBY"
            )}
          </div>
          <Button 
            onClick={handleRefresh} 
            disabled={isLoading || isRefreshing}
            variant="ghost" 
            size="sm" 
            className="h-8 w-8 p-0 hover:bg-zinc-100 dark:hover:bg-zinc-800 text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-all rounded-full"
          >
            <RefreshCcw className={`w-3.5 h-3.5 ${isLoading || isRefreshing ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </div>


      {/* Main Market Switcher */}
      <Tabs defaultValue="vn" className="w-full">
        <div className="flex items-center justify-between mb-6">
          <TabsList className="bg-zinc-100 dark:bg-zinc-900/50 p-1 rounded-xl border border-zinc-200 dark:border-zinc-800/50">
            <TabsTrigger 
              value="vn" 
              className="px-6 py-2 rounded-lg data-[state=active]:bg-emerald-500/10 data-[state=active]:text-emerald-600 dark:data-[state=active]:text-emerald-400 gap-2 transition-all font-bold"
            >
              <Globe className="w-4 h-4" /> VIETNAM MARKET
            </TabsTrigger>
            <TabsTrigger 
              value="us" 
              className="px-6 py-2 rounded-lg data-[state=active]:bg-indigo-500/10 data-[state=active]:text-indigo-600 dark:data-[state=active]:text-indigo-400 gap-2 transition-all font-bold"
            >
              <Globe className="w-4 h-4" /> US MARKETS
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="us" className="space-y-8 animate-in fade-in zoom-in-95 duration-500">
          <IntelligenceBanner 
            scenarios={data?.us?.scenarios} 
            capitalFlow={data?.us?.capitalFlow}
            market="US"
            timeframe={timeframe}
          />

          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pb-4">
            {/* Main Charts area */}
            <div className="lg:col-span-8 space-y-6">
              <Card className={`${GLASS_CARD} border-zinc-700/30 group`}>
                <div className="absolute inset-0 bg-gradient-to-br from-indigo-500/5 via-transparent to-transparent pointer-events-none" />
                <CardHeader className="pb-2 border-b border-zinc-800/50 flex flex-row items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Globe className="w-3.5 h-3.5 text-indigo-400" />
                    <span className="text-[11px] font-bold tracking-widest text-zinc-400 uppercase">Velocity Spectrum 🇺🇸</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-[9px] font-mono border-zinc-800 text-zinc-500 uppercase">{timeframe.toUpperCase()} Active</Badge>
                    <AIDataInsight
                      type="Velocity Bar Chart"
                      description="Bar chart showing percentage change of US market assets. Green bars = positive momentum, Red bars = negative. Measures asset velocity relative to the selected timeframe."
                      data={data?.us?.assets || []}
                      market="US"
                      timeframe={timeframe}
                    />
                  </div>
                </CardHeader>
                <CardContent className="pt-8">
                  <div className="h-[280px] w-full">
                    <MarketBarChart data={data?.us?.assets || []} />
                  </div>
                </CardContent>
              </Card>

              <Card className={`${GLASS_CARD}`}>
                <CardHeader className="bg-zinc-900/40 border-b border-zinc-800/50 py-3 px-4 flex flex-row items-center justify-between">
                  <CardTitle className="text-xs flex items-center gap-2 font-mono text-zinc-400 uppercase">
                    <Search className="w-3.5 h-3.5" />
                    Snapshot Terminal
                  </CardTitle>
                  <AIDataInsight
                    type="Snapshot Table"
                    description="Table showing US market instruments with price, percentage change, day/week performance, and momentum status (fire/sleep/neutral)."
                    data={data?.us?.assets || []}
                    market="US"
                    timeframe={timeframe}
                  />
                </CardHeader>
                <CardContent className="p-0">
                  <MarketSnapshotTable assets={data?.us?.assets || []} />
                </CardContent>
              </Card>
            </div>

            {/* Side Analytics */}
            <div className="lg:col-span-4 space-y-6">
              <div className="relative">
                <div className="absolute top-3 right-3 z-10">
                  <AIDataInsight
                    type="Correlation Heatmap"
                    description="Pairwise correlation matrix for US market assets. Values range from -1 (inverse) to +1 (perfectly correlated). High correlations (>0.85) indicate concentration risk."
                    data={{ assets: data?.us?.assetList || [], matrix: data?.us?.correlationMatrix || [] }}
                    market="US"
                    timeframe={timeframe}
                  />
                </div>
                <CorrelationHeatmap 
                  title="US Heatmatrix" 
                  assets={data?.us?.assetList || []} 
                  matrix={data?.us?.correlationMatrix || []} 
                />
              </div>
              
              <Card className={`${GLASS_CARD} p-4 space-y-4`}>
                <div className="flex items-center justify-between pb-2 border-b border-zinc-800/50">
                  <div className="flex items-center gap-2 text-zinc-400 text-[10px] font-mono uppercase tracking-widest">
                    <Zap className="w-3 h-3 text-orange-400" />
                    Dominant Driver
                  </div>
                  {data?.us?.drivers?.capitalFlowSignal && (
                    <Badge variant="outline" className={`text-[8px] font-mono border-none ${
                      data.us.drivers.capitalFlowSignal === 'RISK-ON' ? 'bg-emerald-500/10 text-emerald-500' :
                      data.us.drivers.capitalFlowSignal === 'DEFENSIVE' ? 'bg-rose-500/10 text-rose-500' :
                      'bg-amber-500/10 text-amber-500'
                    }`}>
                      {data.us.drivers.capitalFlowSignal}
                    </Badge>
                  )}
                </div>
                <div className="space-y-4">
                  <div className="space-y-2">
                    <div className="text-lg font-bold tracking-tight text-white leading-tight">
                      {data?.us?.drivers?.summaryEn || "..."}
                    </div>
                    <div className="text-xs text-zinc-500 italic font-mono leading-relaxed">
                      {data?.us?.drivers?.summaryVi}
                    </div>
                  </div>

                  {data?.us?.drivers?.correlationSignalEn && (
                    <div className="pt-3 border-t border-zinc-800/30 space-y-2">
                      <div className="flex items-center gap-1.5 text-[9px] font-bold text-amber-500 uppercase tracking-tighter">
                        <AlertCircle className="w-3 h-3" /> Correlation Signal
                      </div>
                      <p className="text-[11px] text-zinc-400 leading-snug font-mono italic">
                        {data.us.drivers.correlationSignalEn}
                      </p>
                      <p className="text-[10px] text-zinc-600 leading-snug font-mono italic">
                        {data.us.drivers.correlationSignalVi}
                      </p>
                    </div>
                  )}
                </div>
              </Card>
            </div>
          </div>

          <div className="pt-6 border-t border-zinc-200 dark:border-zinc-800/50 mt-10">
            <TechnicalAnalysisView technicals={data?.us?.technicals} market="US" />
            <AssetValuationTerminal valuation={data?.us?.valuation} market="US" />
          </div>
        </TabsContent>

        <TabsContent value="vn" className="space-y-8 animate-in fade-in zoom-in-95 duration-500">
          <IntelligenceBanner 
            scenarios={data?.vn?.scenarios} 
            capitalFlow={data?.vn?.capitalFlow}
            market="VN"
            timeframe={timeframe}
          />

          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pb-4">
            <div className="lg:col-span-8 space-y-6">
              <Card className={`${GLASS_CARD} border-zinc-700/30 group`}>
                <div className="absolute inset-0 bg-gradient-to-br from-emerald-500/5 via-transparent to-transparent pointer-events-none" />
                <CardHeader className="pb-2 border-b border-zinc-800/50 flex flex-row items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Globe className="w-3.5 h-3.5 text-emerald-400" />
                    <span className="text-[11px] font-bold tracking-widest text-zinc-400 uppercase">Velocity Spectrum 🇻🇳</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge variant="outline" className="text-[9px] font-mono border-zinc-800 text-zinc-500 uppercase">{timeframe.toUpperCase()} Active</Badge>
                    <AIDataInsight
                      type="Velocity Bar Chart"
                      description="Bar chart showing percentage change of Vietnam market assets. Green bars = positive momentum, Red bars = negative. Measures asset velocity relative to the selected timeframe."
                      data={data?.vn?.assets || []}
                      market="VN"
                      timeframe={timeframe}
                    />
                  </div>
                </CardHeader>
                <CardContent className="pt-8">
                  <div className="h-[280px] w-full">
                    <MarketBarChart data={data?.vn?.assets || []} />
                  </div>
                </CardContent>
              </Card>

              <Card className={`${GLASS_CARD}`}>
                <CardHeader className="bg-zinc-900/40 border-b border-zinc-800/50 py-3 px-4 flex flex-row items-center justify-between">
                  <CardTitle className="text-xs flex items-center gap-2 font-mono text-zinc-400 uppercase">
                    <Search className="w-3.5 h-3.5" />
                    Snapshot Terminal
                  </CardTitle>
                  <AIDataInsight
                    type="Snapshot Table"
                    description="Table showing Vietnam market instruments with price, percentage change, day/week performance, and momentum status."
                    data={data?.vn?.assets || []}
                    market="VN"
                    timeframe={timeframe}
                  />
                </CardHeader>
                <CardContent className="p-0">
                  <MarketSnapshotTable assets={data?.vn?.assets || []} />
                </CardContent>
              </Card>
            </div>

            <div className="lg:col-span-4 space-y-6">
              <div className="relative">
                <div className="absolute top-3 right-3 z-10">
                  <AIDataInsight
                    type="Correlation Heatmap"
                    description="Pairwise correlation matrix for Vietnam market assets. Values range from -1 (inverse) to +1 (perfectly correlated). High correlations (>0.85) indicate concentration risk."
                    data={{ assets: data?.vn?.assetList || [], matrix: data?.vn?.correlationMatrix || [] }}
                    market="VN"
                    timeframe={timeframe}
                  />
                </div>
                <CorrelationHeatmap 
                  title="VN Heatmatrix" 
                  assets={data?.vn?.assetList || []} 
                  matrix={data?.vn?.correlationMatrix || []} 
                />
              </div>

              <Card className={`${GLASS_CARD} p-4 space-y-4`}>
                <div className="flex items-center justify-between pb-2 border-b border-zinc-800/50">
                  <div className="flex items-center gap-2 text-zinc-400 text-[10px] font-mono uppercase tracking-widest">
                    <Zap className="w-3 h-3 text-emerald-400" />
                    Dominant Driver
                  </div>
                  {data?.vn?.drivers?.capitalFlowSignal && (
                    <Badge variant="outline" className={`text-[8px] font-mono border-none ${
                      data.vn.drivers.capitalFlowSignal === 'RISK-ON' ? 'bg-emerald-500/10 text-emerald-500' :
                      data.vn.drivers.capitalFlowSignal === 'DEFENSIVE' ? 'bg-rose-500/10 text-rose-500' :
                      'bg-amber-500/10 text-amber-500'
                    }`}>
                      {data.vn.drivers.capitalFlowSignal}
                    </Badge>
                  )}
                </div>
                <div className="space-y-4">
                  <div className="space-y-2">
                    <div className="text-lg font-bold tracking-tight text-white leading-tight">
                      {data?.vn?.drivers?.summaryEn || "..."}
                    </div>
                    <div className="text-xs text-zinc-500 italic font-mono leading-relaxed">
                      {data?.vn?.drivers?.summaryVi}
                    </div>
                  </div>

                  {data?.vn?.drivers?.correlationSignalEn && (
                    <div className="pt-3 border-t border-zinc-800/30 space-y-2">
                      <div className="flex items-center gap-1.5 text-[9px] font-bold text-emerald-500 uppercase tracking-tighter">
                        <AlertCircle className="w-3 h-3" /> Correlation Signal
                      </div>
                      <p className="text-[11px] text-zinc-400 leading-snug font-mono italic">
                        {data.vn.drivers.correlationSignalEn}
                      </p>
                      <p className="text-[10px] text-zinc-600 leading-snug font-mono italic">
                        {data.vn.drivers.correlationSignalVi}
                      </p>
                    </div>
                  )}
                </div>
              </Card>
            </div>
          </div>

          <div className="pt-6 border-t border-zinc-200 dark:border-zinc-800/50 mt-10">
            <TechnicalAnalysisView technicals={data?.vn?.technicals} market="VN" />
            <AssetValuationTerminal valuation={data?.vn?.valuation} market="VN" />
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function IntelligenceBanner({ scenarios, capitalFlow, market }: { scenarios?: any[], capitalFlow: any, market: string, timeframe: string }) {
  if (!scenarios || scenarios.length === 0 || !capitalFlow) return null;

  const isUS = market === 'US';
  const accent = isUS ? 'indigo' : 'emerald';

  return (
    <div className={`${GLASS_CARD} flex flex-col divide-y divide-zinc-200 dark:divide-zinc-800/50 p-0 mb-8 rounded-2xl overflow-hidden`}>
      {/* 3 Scenario Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 divide-y md:divide-y-0 md:divide-x divide-zinc-200 dark:divide-zinc-800/50">
        {scenarios.map((scenario, idx) => {
          const regimeColor = scenario.regime === 'Risk-ON' ? 'text-emerald-400' : 
                             scenario.regime === 'Crisis' ? 'text-rose-400' : 
                             scenario.regime === 'Goldilocks' ? 'text-indigo-400' :
                             scenario.regime === 'Stagflation' ? 'text-amber-400' :
                             'text-zinc-400';
          
          return (
            <div key={idx} className="p-5 space-y-3 relative group hover:bg-zinc-50/50 dark:hover:bg-zinc-900/20 transition-colors">
              <div className="flex items-center justify-between">
                <span className="text-[9px] font-bold tracking-widest text-zinc-500 uppercase italic">
                  Scenario {idx + 1}
                </span>
                <Badge 
                  variant="outline" 
                  className={`text-[9px] font-mono border-none scale-90 ${
                    scenario.confidence >= 80 ? 'bg-emerald-500/10 text-emerald-500' :
                    scenario.confidence >= 50 ? 'bg-amber-500/10 text-amber-500' :
                    'bg-rose-500/10 text-rose-500'
                  }`}
                >
                  {scenario.confidence}% CONF
                </Badge>
              </div>
              <div className="space-y-1">
                <h3 className={`text-sm font-bold tracking-tight uppercase ${regimeColor}`}>
                  {scenario.name}
                </h3>
                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-relaxed line-clamp-3">
                  {scenario.summaryEn}
                </p>
              </div>
              <p className="text-[10px] text-zinc-400 dark:text-zinc-500 italic font-mono border-l pl-2 border-zinc-200 dark:border-zinc-800/50 line-clamp-2">
                {scenario.summaryVi}
              </p>
            </div>
          );
        })}
      </div>

      {/* Smart Money Flow Section */}
      <div className="p-5 flex flex-col md:flex-row items-center gap-6 bg-zinc-50/50 dark:bg-zinc-950/40">
        <div className="flex items-center gap-4 min-w-[200px]">
          <div className="relative">
            <Shield className="w-8 h-8 text-zinc-200 dark:text-zinc-800" />
            <div className={`absolute inset-0 flex items-center justify-center`}>
              {capitalFlow.signal === 'RISK-ON' ? (
                <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse ring-4 ring-emerald-500/20" />
              ) : capitalFlow.signal === 'DEFENSIVE' ? (
                <div className="w-2.5 h-2.5 rounded-full bg-rose-500 ring-4 ring-rose-500/20" />
              ) : (
                <div className="w-2.5 h-2.5 rounded-full bg-amber-500 ring-4 ring-amber-500/20" />
              )}
            </div>
          </div>
          <div>
            <div className="text-[10px] font-bold tracking-[0.2em] text-zinc-500 uppercase font-mono mb-0.5">Smart Money Flow</div>
            <div className={`text-xl font-black tracking-tighter ${TERMINAL_FONT}`}>
              {capitalFlow.signal}
            </div>
          </div>
        </div>
        
        <div className="flex-1 space-y-1">
          <p className="text-xs text-zinc-600 dark:text-zinc-300 font-medium leading-relaxed">
            {capitalFlow.summaryEn}
          </p>
          <p className="text-[10px] text-zinc-500 italic font-mono">
            {capitalFlow.summaryVi}
          </p>
        </div>

        <div className="hidden lg:flex items-center gap-2 px-4 py-2 rounded-xl bg-white/50 dark:bg-zinc-900/50 border border-zinc-200/50 dark:border-zinc-800/50">
          <TrendingUp className={`w-3.5 h-3.5 ${capitalFlow.signal === 'RISK-ON' ? 'text-emerald-500' : 'text-zinc-400'}`} />
          <span className="text-[9px] font-bold font-mono text-zinc-500 uppercase tracking-tighter">Institutional Tracking Active</span>
        </div>
      </div>
    </div>
  );
}

function MarketBarChart({ data }: { data: MarketAsset[] }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  if (!data.length) return <div className="flex items-center justify-center h-full text-zinc-400 dark:text-zinc-600 font-mono text-[10px]">AWAITING STREAM...</div>;

  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
        <defs>
          <linearGradient id="barUp" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#10b981" stopOpacity={0.8}/>
            <stop offset="95%" stopColor="#10b981" stopOpacity={0.1}/>
          </linearGradient>
          <linearGradient id="barDown" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#f43f5e" stopOpacity={0.8}/>
            <stop offset="95%" stopColor="#f43f5e" stopOpacity={0.1}/>
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke={isDark ? "#18181b" : "#f4f4f5"} />
        <XAxis 
          dataKey="name" 
          stroke={isDark ? "#3f3f46" : "#d4d4d8"} 
          fontSize={10} 
          tickLine={false} 
          axisLine={false}
          interval={0}
          tick={{ fill: isDark ? '#52525b' : '#a1a1aa', fontWeight: 600 }}
        />
        <YAxis 
          stroke={isDark ? "#3f3f46" : "#d4d4d8"} 
          fontSize={10} 
          tickLine={false} 
          axisLine={false}
          unit="%"
          tick={{ fill: isDark ? '#52525b' : '#a1a1aa' }}
        />
        <Tooltip 
          contentStyle={{ 
            backgroundColor: isDark ? "rgba(9, 9, 11, 0.95)" : "rgba(255, 255, 255, 0.95)", 
            backdropFilter: "blur(12px)",
            border: isDark ? "1px solid rgba(63, 63, 70, 0.4)" : "1px solid rgba(228, 228, 231, 0.8)", 
            borderRadius: "12px",
            fontSize: "10px", 
            color: isDark ? "#fff" : "#18181b",
            boxShadow: "0 20px 25px -5px rgba(0, 0, 0, 0.1)"
          }}
          cursor={{ fill: isDark ? 'rgba(255, 255, 255, 0.02)' : 'rgba(0, 0, 0, 0.02)' }}
          formatter={(value: any) => [`${Number(value) > 0 ? "+" : ""}${Number(value).toFixed(2)}%`, 'VELOCITY']}
        />
        <ReferenceLine y={0} stroke={isDark ? "#27272a" : "#e4e4e7"} strokeWidth={1} />
        <Bar dataKey="percentChange" radius={[4, 4, 0, 0]}>
          {data.map((entry, index) => (
            <Cell 
              key={`cell-${index}`} 
              fill={entry.percentChange >= 0 ? "url(#barUp)" : "url(#barDown)"} 
            />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}

function CorrelationHeatmap({ title, assets, matrix }: { title: string, assets: string[], matrix: number[][] }) {
  const { theme } = useTheme();
  const isDark = theme === 'dark';

  if (!assets.length || !matrix.length) return null;

  const getColor = (val: number) => {
    if (val > 0) return `rgba(16, 185, 129, ${val * (isDark ? 0.7 : 0.6)})`; 
    if (val < 0) return `rgba(244, 63, 94, ${Math.abs(val) * (isDark ? 0.7 : 0.6)})`; 
    return isDark ? "rgba(24, 24, 27, 0.5)" : "rgba(244, 244, 245, 0.8)";
  };

  return (
    <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/30`}>
      <CardHeader className="bg-zinc-50 dark:bg-zinc-950/40 border-b border-zinc-200 dark:border-zinc-800/50 py-3 px-4 flex flex-row items-center justify-between">
        <CardTitle className="text-[9px] uppercase font-bold tracking-[0.3em] text-zinc-400 dark:text-zinc-500 font-mono">
          {title}
        </CardTitle>
        <TooltipProvider>
          <ShadcnTooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" className="h-4 w-4 p-0 text-zinc-400 hover:text-indigo-400">
                <Info className="w-3 h-3" />
              </Button>
            </TooltipTrigger>
            <TooltipContent side="left" className="w-64 p-3 bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 shadow-2xl">
              <div className="space-y-3">
                <div className="space-y-1">
                  <h4 className="text-[10px] font-bold uppercase text-indigo-500">How to Read</h4>
                  <p className="text-[11px] leading-relaxed text-zinc-600 dark:text-zinc-400">
                    Positive correlation means assets move together; negative means opposite. 
                    A broken correlation alert signals a market regime shift.
                  </p>
                </div>
                <div className="space-y-1 border-t border-zinc-100 dark:border-zinc-800/50 pt-2 text-zinc-500 italic">
                  <h4 className="text-[10px] font-bold uppercase text-emerald-500">Cách đọc</h4>
                  <p className="text-[11px] leading-relaxed">
                    Tương quan dương (+) là cùng chiều, âm (-) là ngược chiều. 
                    Cảnh báo "broken correlation" là dấu hiệu chuyển đổi trạng thái thị trường.
                  </p>
                </div>
              </div>
            </TooltipContent>
          </ShadcnTooltip>
        </TooltipProvider>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto">
          <table className="w-full border-collapse">
            <thead>
              <tr>
                <th className="p-2 border-b border-r border-zinc-200 dark:border-zinc-800/50 bg-zinc-100 dark:bg-zinc-950/60"></th>
                {assets.map((a, i) => (
                  <th key={i} className="p-2 border-b border-zinc-200 dark:border-zinc-800/50 text-[8px] font-mono text-zinc-400 dark:text-zinc-600 text-center uppercase tracking-tighter w-10 min-w-[36px]">
                    {a.slice(0, 5)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {matrix.map((row, i) => (
                <tr key={i} className="group/row hover:bg-black/[0.02] dark:hover:bg-white/[0.03] transition-colors">
                  <td className="p-2 border-r border-zinc-200 dark:border-zinc-800/50 text-[8px] font-bold uppercase bg-zinc-100 dark:bg-zinc-950/60 text-zinc-500 group-hover/row:text-zinc-900 dark:group-hover/row:text-zinc-200 transition-colors">
                    {assets[i]}
                  </td>
                  {row.map((val, j) => (
                    <TooltipProvider key={j}>
                      <ShadcnTooltip>
                        <TooltipTrigger asChild>
                          <td 
                            className="p-1 text-center text-[9px] font-mono border border-zinc-200 dark:border-zinc-900/40 cursor-crosshair transition-all hover:brightness-110 active:scale-95"
                            style={{ 
                              backgroundColor: getColor(val),
                              color: Math.abs(val) > 0.6 ? (isDark ? "white" : "#000") : (isDark ? "rgba(255,255,255,0.4)" : "rgba(0,0,0,0.5)")
                            }}
                          >
                            {val === 1 ? "1.0" : val.toFixed(2)}
                          </td>
                        </TooltipTrigger>
                        <TooltipContent side="top" className="bg-white dark:bg-zinc-950 border-zinc-200 dark:border-zinc-800 text-[10px] font-mono p-2 rounded-lg shadow-2xl">
                           <div className="font-bold text-zinc-900 dark:text-zinc-300">{assets[i]} × {assets[j]}</div>
                           <div className={`mt-0.5 ${val > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'}`}>COEFF: {val.toFixed(3)}</div>
                        </TooltipContent>
                      </ShadcnTooltip>
                    </TooltipProvider>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  );
}

function MarketSnapshotTable({ assets }: { assets: MarketAsset[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-[11px] font-mono border-collapse tabular-nums">
        <thead>
          <tr className="bg-zinc-100 dark:bg-zinc-950/60 text-zinc-500 dark:text-zinc-500 uppercase tracking-widest text-[9px] border-b border-zinc-200 dark:border-zinc-800/50">
            <th className="text-left py-4 px-6 font-bold">INSTRUMENT</th>
            <th className="text-right py-4 px-4 font-bold">SIGNAL</th>
            <th className="text-right py-4 px-4 font-bold">Δ%</th>
            <th className="text-right py-4 px-4 font-bold">DAY</th>
            <th className="text-right py-4 px-4 font-bold">WEEK</th>
            <th className="text-center py-4 px-6 font-bold">STATUS</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200 dark:divide-zinc-800/20">
          {assets.map((a, i) => (
            <tr key={i} className="hover:bg-zinc-50 dark:hover:bg-white/[0.03] transition-all group">
              <td className="py-3 px-6 border-r border-zinc-200 dark:border-zinc-800/30">
                <div className="flex flex-col">
                  <span className="font-bold text-zinc-700 dark:text-zinc-200 group-hover:text-emerald-600 dark:group-hover:text-emerald-400 transition-colors uppercase">{a.name}</span>
                  <span className="text-[9px] text-zinc-400 dark:text-zinc-600 font-bold opacity-50">{a.symbol}</span>
                </div>
              </td>
              <td className="py-3 px-4 text-right font-medium text-zinc-600 dark:text-zinc-300">
                {a.market === 'VN' && a.symbol !== 'USD/VND' ? 
                  new Intl.NumberFormat('vi-VN').format(a.price) : 
                  new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 2 }).format(a.price)}
              </td>
              <td className={`py-3 px-4 text-right font-bold ${a.percentChange >= 0 ? "text-emerald-400" : "text-rose-400"}`}>
                <div className="flex items-center justify-end gap-1">
                  {a.percentChange > 0 ? "+" : ""}{a.percentChange.toFixed(2)}
                  <span className="text-[9px] opacity-70">%</span>
                </div>
              </td>
              <td className={`py-3 px-4 text-right opacity-60 ${a.dayChange >= 0 ? "text-emerald-500" : "text-rose-500"}`}>
                {a.dayChange.toFixed(2)}%
              </td>
              <td className={`py-3 px-4 text-right opacity-40 ${a.weekChange >= 0 ? "text-emerald-600" : "text-rose-600"}`}>
                {a.weekChange.toFixed(2)}%
              </td>
              <td className="py-3 px-6 text-center">
                <div className="flex items-center justify-center gap-2">
                  {a.momentum === 'fire' ? (
                    <div className="p-1 rounded bg-orange-500/10"><Flame className="w-3 h-3 text-orange-500 animate-pulse" /></div>
                  ) : a.momentum === 'sleep' ? (
                    <Moon className="w-3 h-3 text-zinc-800" />
                  ) : (
                    a.direction === 'up' ? 
                      <ArrowUpRight className="w-3 h-3 text-emerald-500/50" /> : 
                      <ArrowDownRight className="w-3 h-3 text-rose-500/50" />
                  )}
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TechnicalAnalysisView({ technicals, market }: { technicals?: any, market: string }) {
  if (!technicals) return null;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-2 mb-2">
        <div className="flex items-center gap-2">
          <Compass className={`w-5 h-5 ${market === 'US' ? 'text-indigo-500' : 'text-emerald-500'}`} />
          <h3 className="text-lg font-bold tracking-tight uppercase">Technical Spectrum: {market}</h3>
        </div>
        <AIDataInsight
          type="Technical Analysis Suite"
          description={`Comprehensive technical analysis for the ${market} market, including market cycles, key support/resistance levels, and historical seasonality patterns.`}
          data={technicals}
          market={market}
          timeframe="1D" // Generally Technical Analysis reflects larger timeframe trends
        />
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Cycle Analysis Card */}
        <Card className={`${GLASS_CARD} p-5 border-zinc-200 dark:border-zinc-800/50`}>
          <div className="flex items-center justify-between mb-4">
            <span className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">Market Cycle Phase</span>
            <Badge className={`${
              technicals.cycle.phase === 'Markup' ? 'bg-emerald-500/10 text-emerald-500' :
              technicals.cycle.phase === 'Decline' ? 'bg-rose-500/10 text-rose-500' :
              'bg-blue-500/10 text-blue-500'
            } border-none`}>
              {technicals.cycle.phase.toUpperCase()}
            </Badge>
          </div>
          <div className="space-y-2">
            <div className="text-xl font-bold">{technicals.cycle.phase} Phase</div>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 leading-relaxed">
              {technicals.cycle.description}
            </p>
            <p className="text-[10px] text-zinc-400 italic">
              {technicals.cycle.descriptionVi}
            </p>
            <div className="pt-4">
              <div className="flex items-center justify-between text-[10px] mb-1 font-mono uppercase">
                <span>Phase Strength</span>
                <span>{technicals.cycle.strength.toFixed(1)}%</span>
              </div>
              <div className="h-1.5 w-full bg-zinc-100 dark:bg-zinc-800 rounded-full overflow-hidden">
                <div 
                  className={`h-full ${market === 'US' ? 'bg-indigo-500' : 'bg-emerald-500'}`} 
                  style={{ width: `${technicals.cycle.strength}%` }}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Support/Resistance */}
        <Card className={`${GLASS_CARD} p-0 overflow-hidden border-zinc-200 dark:border-zinc-800/50`}>
          <div className="bg-zinc-50 dark:bg-zinc-800/20 p-4 border-b border-zinc-100 dark:border-zinc-800/50">
            <div className="text-[10px] font-bold uppercase tracking-widest text-zinc-500">Key Levels (S/R)</div>
          </div>
          <div className="p-4 space-y-4">
            {technicals.supportResistance.slice(0, 2).map((item: any, idx: number) => (
              <div key={idx} className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-xs font-bold font-mono">{item.symbol}</span>
                  <span className="text-[10px] text-zinc-400">BOLLINGER WIDTH: {(((item.bollingerUpper - item.bollingerLower) / item.bollingerMid) * 100).toFixed(1)}%</span>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="bg-rose-500/5 p-2 rounded border border-rose-500/10">
                    <div className="text-[9px] uppercase text-rose-500 font-bold mb-1">Resistance</div>
                    <div className="flex flex-wrap gap-1">
                      {item.resistance.map((r: number, i: number) => (
                        <span key={i} className="text-[10px] font-mono">{r.toLocaleString()}</span>
                      ))}
                    </div>
                  </div>
                  <div className="bg-emerald-500/5 p-2 rounded border border-emerald-500/10">
                    <div className="text-[9px] uppercase text-emerald-500 font-bold mb-1">Support</div>
                    <div className="flex flex-wrap gap-1">
                      {item.support.map((s: number, i: number) => (
                        <span key={i} className="text-[10px] font-mono">{s.toLocaleString()}</span>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </Card>
      </div>

      {/* Seasonality Chart */}
      <Card className={`${GLASS_CARD} p-5 border-zinc-200 dark:border-zinc-800/50`}>
        <div className="text-[10px] font-bold uppercase tracking-widest text-zinc-500 mb-4">Historical Seasonality: {market}</div>
        <div className="h-[150px] w-full mt-2">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={technicals.seasonality.filter((s:any) => s.type === 'day')}>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(128,128,128,0.1)" />
              <XAxis dataKey="label" fontSize={9} tickLine={false} axisLine={false} />
              <YAxis hide />
              <Tooltip 
                contentStyle={{ backgroundColor: 'rgba(0,0,0,0.8)', border: 'none', borderRadius: '8px', fontSize: '10px' }}
                itemStyle={{ color: '#fff' }}
              />
              <Bar dataKey="value" radius={[2, 2, 0, 0]}>
                {technicals.seasonality.filter((s:any) => s.type === 'day').map((entry: any, index: number) => (
                  <Cell key={`cell-${index}`} fill={entry.value >= 0 ? (market === 'US' ? '#6366f1' : '#10b981') : '#f43f5e'} fillOpacity={0.6} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        </div>
      </Card>
    </div>
  );
}

function AssetValuationTerminal({ valuation, market }: { valuation?: any, market: string }) {
  if (!valuation) return null;

  const accentColor = market === 'US' ? 'indigo' : 'emerald';
  const accentHex = market === 'US' ? '#6366f1' : '#10b981';

  return (
    <div className="pt-8 border-t border-zinc-100 dark:border-zinc-800/50 mt-10">
      <div className="flex items-center justify-between gap-2 mb-6">
        <div className="flex items-center gap-2">
          <Layers className={`w-5 h-5 ${market === 'US' ? 'text-indigo-500' : 'text-emerald-500'}`} />
          <h3 className="text-xl font-bold tracking-tight uppercase">Valuation Decision Terminal: {market}</h3>
        </div>
        <AIDataInsight
          type="Valuation Terminal"
          description={`Financial valuation models for the ${market} market, featuring DCF projections, Monte Carlo probability distributions, and competitive sector benchmarking.`}
          data={valuation}
          market={market}
          timeframe="1D" // Valuation models span larger horizons
        />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* DCF Valuation */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50`}>
          <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
            <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
              <Binary className="w-3.5 h-3.5" /> DCF Model Scenarios
            </CardTitle>
          </CardHeader>
          <CardContent className="p-5 space-y-4">
            {valuation.dcf.map((item: any, idx: number) => (
              <div key={idx} className="space-y-3 pb-4 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-sm tracking-tight">{item.symbol}</span>
                  <Badge className={`${item.upside > 0 ? 'bg-emerald-500/10 text-emerald-500' : 'bg-rose-500/10 text-rose-500'} border-none uppercase text-[9px]`}>
                    {item.upside > 0 ? '+' : ''}{item.upside.toFixed(1)}% Potential
                  </Badge>
                </div>
                <div className="flex items-baseline gap-2">
                  <span className="text-xl font-mono font-bold tracking-tighter">
                    {market === 'VN' && item.symbol !== 'USD/VND' ? item.fairValue.toLocaleString() : `$${item.fairValue.toFixed(2)}`}
                  </span>
                  <span className="text-[10px] text-zinc-400 font-mono uppercase tracking-widest font-bold">Estimated FV</span>
                </div>
                <div className="grid grid-cols-1 gap-1.5">
                  {item.assumptions.map((ass: any, i: number) => (
                    <div key={i} className="flex items-center justify-between text-[10px] font-mono">
                      <span className="text-zinc-500">{ass.label}</span>
                      <span className="text-zinc-300 font-bold">{ass.value}</span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </CardContent>
        </Card>

        {/* Monte Carlo Probability Area */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50`}>
          <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
            <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
              <BarChart3 className="w-3.5 h-3.5" /> Monte Carlo Simulation
            </CardTitle>
          </CardHeader>
          <CardContent className="p-5 space-y-8 max-h-[500px] overflow-y-auto custom-scrollbar">
             {valuation.monteCarlo.map((sim: any, idx: number) => (
               <div key={idx} className="space-y-6 pb-6 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0 animate-in fade-in slide-in-from-bottom-2 duration-300">
                 <div className="flex items-center justify-between">
                   <h4 className="text-xs font-bold font-mono text-zinc-400 uppercase tracking-tighter">
                     <span className={`text-${accentColor}-500 mr-2`}>▶</span>
                     {sim.symbol} Probabilities
                   </h4>
                   <span className="text-[9px] text-zinc-500 font-mono">n={sim.iterations.toLocaleString()} PATHS</span>
                 </div>
                 <div className="relative h-20 flex items-center justify-between px-2">
                   <div className="absolute inset-x-0 h-1 bg-gradient-to-r from-rose-500/20 via-emerald-500/30 to-rose-500/20 rounded-full" />
                   <div className="relative flex flex-col items-center">
                     <span className="text-[8px] font-bold text-rose-500 mb-3 bg-rose-500/10 px-1 rounded uppercase">P10</span>
                     <div className="w-1.5 h-1.5 rounded-full bg-rose-500 border border-white dark:border-zinc-900 shadow-lg" />
                     <span className="mt-1.5 font-mono text-[9px] font-bold">{sim.p10.toLocaleString()}</span>
                   </div>
                   <div className="relative flex flex-col items-center">
                     <span className="text-[8px] font-bold text-emerald-500 mb-3 bg-emerald-500/10 px-1 rounded uppercase">P50</span>
                     <div className="w-2.5 h-2.5 rounded-full bg-emerald-500 border-2 border-white dark:border-zinc-900 shadow-lg" />
                     <span className="mt-1.5 font-mono text-[10px] font-bold">{sim.p50.toLocaleString()}</span>
                   </div>
                   <div className="relative flex flex-col items-center">
                     <span className="text-[8px] font-bold text-indigo-500 mb-3 bg-indigo-500/10 px-1 rounded uppercase">P90</span>
                     <div className="w-1.5 h-1.5 rounded-full bg-indigo-500 border border-white dark:border-zinc-900 shadow-lg" />
                     <span className="mt-1.5 font-mono text-[9px] font-bold">{sim.p90.toLocaleString()}</span>
                   </div>
                 </div>
               </div>
             ))}
             {valuation.monteCarlo.length > 0 && (
               <div className="pt-2 text-[10px] text-zinc-500 leading-relaxed italic text-center p-3 rounded-lg bg-zinc-50 dark:bg-zinc-800/20 border border-dashed border-zinc-200 dark:border-zinc-700/50">
                  Proprietary simulation showing range of probable outcomes over 10,000 algorithmic iterations.
               </div>
             )}
          </CardContent>
        </Card>

        {/* Sector Comparison */}
        <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 overflow-hidden`}>
           <CardHeader className="py-4 px-5 border-b border-zinc-100 dark:border-zinc-800/30 bg-zinc-50/50 dark:bg-zinc-900/40">
             <CardTitle className="text-xs font-mono font-bold uppercase tracking-widest text-zinc-500 flex items-center gap-2">
               <PieChart className="w-3.5 h-3.5" /> Sector Benchmarking
             </CardTitle>
           </CardHeader>
            <CardContent className="p-0 max-h-[500px] overflow-y-auto custom-scrollbar">
              <div className="p-5 space-y-8">
                {valuation.sectorComparison.map((sec: any, idx: number) => (
                  <div key={idx} className="space-y-4 pb-6 border-b border-zinc-100 dark:border-zinc-800/30 last:border-0 last:pb-0 animate-in fade-in slide-in-from-bottom-2 duration-300">
                    <div className="flex items-center justify-between">
                      <div className="text-xs font-bold font-mono uppercase text-zinc-400 tracking-tighter">
                        <span className={`text-${accentColor}-500 mr-2`}>▶</span>
                        {sec.symbol} vs. Peers
                      </div>
                      <Badge variant="outline" className="text-[8px] uppercase font-mono border-zinc-200 dark:border-zinc-800 scale-90">{sec.sector}</Badge>
                    </div>
                    <div className="space-y-3 pt-2">
                      {sec.metrics.map((m: any, i: number) => (
                        <div key={i} className="space-y-1.5">
                          <div className="flex items-center justify-between text-[9px] uppercase font-bold tracking-tight">
                            <span className="text-zinc-500">{m.label}</span>
                            <span className={m.asset < m.avg ? "text-emerald-500" : "text-zinc-500"}>{m.asset} <span className="opacity-50 mx-1">/</span> {m.avg}</span>
                          </div>
                          <div className="relative h-1.5 w-full bg-zinc-100 dark:bg-zinc-800 rounded-full overflow-hidden">
                             <div className="absolute inset-y-0 left-0 bg-zinc-400 opacity-20" style={{ width: `${(Math.min(m.avg, 100) / (Math.max(m.asset, m.avg, 1) * 1.2)) * 100}%` }} />
                             <div className={`absolute inset-y-0 left-0 ${market === 'US' ? 'bg-indigo-500' : 'bg-emerald-500'}`} style={{ width: `${(Math.min(m.asset, 100) / (Math.max(m.asset, m.avg, 1) * 1.2)) * 100}%` }} />
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
        </Card>
      </div>
    </div>
  );
}
