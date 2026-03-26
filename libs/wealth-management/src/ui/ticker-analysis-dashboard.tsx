'use client';

import { useState } from 'react';
import { Card, CardContent } from './card';
import { Badge } from './badge';
import { Button } from './button';
import { Input } from './input';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './tabs';
import { Loader2, Search, ArrowRight, Zap } from 'lucide-react';
import { TechnicalAnalysisView } from './market-pulse/TechnicalAnalysisView';
import { SeasonalityStatsTable } from './market-pulse/SeasonalityStatsTable';
import { GLASS_CARD } from './market-pulse/constants';
import { fmarketApi } from '@wealth-management/services';
import { TickerValuationView } from './ticker-analysis/TickerValuationView';

export function TickerAnalysisDashboard() {
  const [symbol, setSymbol] = useState('');
  const [isSearching, setIsSearching] = useState(false);
  const [details, setDetails] = useState<any>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [analysisData, setAnalysisData] = useState<any>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSearch = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!symbol.trim()) return;

    setIsSearching(true);
    setDetails(null);
    setAnalysisData(null);
    setError(null);

    try {
      const upperSymbol = symbol.toUpperCase();

      try {
        const fmRes = await fmarketApi.getProductDetails(upperSymbol);
        if (fmRes?.status === 200 && fmRes?.data) {
          const d = fmRes.data;
          setDetails({
            symbol: d.code,
            fullName: d.name,
            description:
              d.description ||
              `Quỹ đầu tư ${d.fundAssetType === 'STOCK_FUND' ? 'cổ phiếu' : 'trái phiếu'} được quản lý bởi ${d.owner?.name || 'tổ chức uy tín'}.`,
            sector: d.fundAssetType === 'STOCK_FUND' ? 'Equity Fund' : 'Bond Fund',
            industry: d.owner?.shortName || 'Fmarket',
            market: 'VN',
            type: 'IFC',
            id: d.id,
          });
          setIsSearching(false);
          return;
        }
      } catch (fmarketErr) {
        console.error('Fmarket API error:', fmarketErr);
      }

      if (upperSymbol === 'IFC') {
        setDetails({
          symbol: 'IFC',
          fullName: 'Investment Fund Certificates',
          description: 'Hệ thống các chứng chỉ quỹ mở tại Việt Nam. Dữ liệu được truy xuất trực tiếp từ cổng Fmarket.',
          sector: 'Financial Services',
          industry: 'Asset Management',
          market: 'VN',
          type: 'IFC',
        });
        setIsSearching(false);
        return;
      }

      const res = await fetch('/api/ticker/details', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ symbol: upperSymbol }),
      });

      if (!res.ok) throw new Error('Failed to fetch ticker details');
      const data = await res.json();
      setDetails(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsSearching(false);
    }
  };

  const handleProceed = async () => {
    if (!details) return;

    setIsAnalyzing(true);
    setError(null);

    try {
      if (details.type === 'IFC' && details.id) {
        const navRes = await fmarketApi.getNavHistory(details.id);
        if (navRes?.status === 200 && navRes?.data) {
          const navs = navRes.data as any[];
          const prices = navs.map((n: any) => n.nav);
          const lastNav = prices[0] || 0;
          const prevNav = prices[1] || lastNav;
          const pctChange = ((lastNav - prevNav) / prevNav) * 100;

          const mockTechnicals = {
            n: navs.length,
            indicators: {
              rsi: 55,
              sma20: lastNav,
              ema20: lastNav * 0.98,
              ema50: lastNav * 0.95,
            },
            trend: {
              direction: pctChange >= 0 ? 'Tăng trưởng' : 'Điều chỉnh',
              directionEn: pctChange >= 0 ? 'Up' : 'Down',
              strength: Math.min(Math.abs(pctChange) * 10 + 50, 95),
              confidence: 85,
            },
            cycle: {
              phase: pctChange >= 0 ? 'Markup' : 'Accumulation',
              descriptionVi: pctChange >= 0 ? 'Giai đoạn tăng trưởng' : 'Giai đoạn tích lũy',
              confidence: 80,
              phases: [
                { label: 'Growth', value: 70, color: '#10b981' },
                { label: 'Value', value: 30, color: '#6366f1' },
              ],
            },
            supportResistance: [
              {
                symbol: details.symbol,
                resistance: [lastNav * 1.05, lastNav * 1.1],
                support: [lastNav * 0.95, lastNav * 0.9],
                bollingerMid: lastNav,
                bollingerUpper: lastNav * 1.05,
                bollingerLower: lastNav * 0.95,
              },
            ],
            seasonality: [],
          };

          setAnalysisData({
            technicals1h: mockTechnicals,
            technicals1d: mockTechnicals,
            valuation: {
              dcf: [
                {
                  symbol: details.symbol,
                  fairValue: lastNav,
                  upside: 0,
                  assumptions: [
                    { label: 'Current NAV', value: lastNav.toLocaleString() },
                    { label: 'Asset Type', value: details.sector },
                  ],
                },
              ],
              monteCarlo: [],
              sectorComparison: [],
            },
          });
          setIsAnalyzing(false);
          return;
        }
      }

      const res = await fetch('/api/ticker/analyze', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          symbol: details.symbol,
          name: details.fullName,
          market: details.market || 'VN',
        }),
      });

      if (!res.ok) throw new Error('Failed to run deep analysis');
      const data = await res.json();
      setAnalysisData(data);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setIsAnalyzing(false);
    }
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div className={`p-6 rounded-3xl ${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50`}>
        <div className="flex flex-col gap-4">
          <form onSubmit={handleSearch} className="flex gap-4 items-center">
            <div className="relative flex-1 max-w-md">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-500" />
              <Input
                value={symbol}
                onChange={(e) => setSymbol(e.target.value)}
                placeholder="Ticker/CCQ (e.g. VCB, DCDS, IFC)"
                className="pl-10 bg-zinc-50 dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800/50 uppercase"
                disabled={isSearching || isAnalyzing}
              />
            </div>
            <Button
              type="submit"
              disabled={!symbol.trim() || isSearching || isAnalyzing}
              className="bg-indigo-600 hover:bg-indigo-700 text-white shadow-lg shadow-indigo-500/20"
            >
              {isSearching ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : <Search className="w-4 h-4 mr-2" />}
              Fetch Details
            </Button>
          </form>
          <div className="px-1">
            <p className="text-[10px] text-zinc-500 font-medium flex items-center gap-1.5 uppercase tracking-tighter">
              <Zap className="w-3 h-3 text-amber-500" />
              Supports Stocks (HSX/HNX) and IFC (Investment Fund Certificates) via Fmarket
            </p>
          </div>

          {error && <div className="text-sm text-rose-500 font-medium bg-rose-500/10 p-4 rounded-xl">{error}</div>}

          {details && !analysisData && (
            <div className="mt-4 p-6 bg-zinc-50 dark:bg-zinc-900/40 rounded-2xl border border-zinc-200 dark:border-zinc-800/50 space-y-4 animate-in slide-in-from-bottom-4">
              <div className="flex items-start justify-between">
                <div>
                  <h3 className="text-2xl font-black tracking-tight">{details.symbol}</h3>
                  <p className="text-sm font-bold text-zinc-500">{details.fullName}</p>
                </div>
                <Badge
                  variant="outline"
                  className={`${details.type === 'IFC' ? 'border-emerald-500/30 text-emerald-400 bg-emerald-500/10' : 'border-indigo-500/30 text-indigo-400 bg-indigo-500/10'} uppercase`}
                >
                  {details.type === 'IFC' ? 'IFC Certificate' : `${details.market} Market`}
                </Badge>
              </div>

              <div className="flex gap-4">
                <Badge variant="secondary" className="bg-zinc-200 dark:bg-zinc-800">
                  Sector: {details.sector}
                </Badge>
                <Badge variant="secondary" className="bg-zinc-200 dark:bg-zinc-800">
                  Industry: {details.industry}
                </Badge>
              </div>

              <p className="text-sm text-zinc-600 dark:text-zinc-400 leading-relaxed border-t border-zinc-200 dark:border-zinc-800/50 pt-4">
                {details.description}
              </p>

              <div className="flex justify-end pt-4">
                <Button
                  onClick={handleProceed}
                  disabled={isAnalyzing}
                  className="bg-emerald-600 hover:bg-emerald-700 text-white shadow-lg shadow-emerald-500/20 px-8"
                >
                  {isAnalyzing ? (
                    <>
                      <Loader2 className="w-4 h-4 mr-2 animate-spin" /> Deep Analyzing...
                    </>
                  ) : (
                    <>
                      Run Deep Analysis <ArrowRight className="w-4 h-4 ml-2" />
                    </>
                  )}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      {analysisData && (
        <div className="space-y-6">
          <div className="flex flex-col gap-1">
            <span className="text-[10px] uppercase font-bold text-zinc-500 tracking-widest">Mã</span>
            <h2 className="text-3xl font-black">{details?.symbol}</h2>
          </div>

          <Tabs defaultValue="cycle" className="space-y-6">
            <TabsList className="bg-transparent border-b border-zinc-800 rounded-none w-full justify-start h-auto p-0 gap-6">
              <TabsTrigger
                value="cycle"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-indigo-500 data-[state=active]:text-indigo-400 px-0 pb-3 text-sm font-bold uppercase tracking-wide bg-transparent"
              >
                Chu kỳ giá
              </TabsTrigger>
              <TabsTrigger
                value="valuation"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-indigo-500 data-[state=active]:text-indigo-400 px-0 pb-3 text-sm font-bold uppercase tracking-wide bg-transparent"
              >
                Mức giá
              </TabsTrigger>
              <TabsTrigger
                value="seasonality"
                className="rounded-none border-b-2 border-transparent data-[state=active]:border-indigo-500 data-[state=active]:text-indigo-400 px-0 pb-3 text-sm font-bold uppercase tracking-wide bg-transparent"
              >
                Mùa Vụ
              </TabsTrigger>
            </TabsList>

            <TabsContent value="cycle" className="space-y-6 mt-6">
              <div className="space-y-4">
                <h4 className="text-sm font-bold text-zinc-400">1H - Khung thời gian</h4>
                <TechnicalAnalysisView
                  technicals={analysisData.technicals1h}
                  market={details?.market || 'VN'}
                  showSeasonality={false}
                />
              </div>
              <div className="space-y-4 mt-8">
                <h4 className="text-sm font-bold text-zinc-400">1D - Khung thời gian</h4>
                <TechnicalAnalysisView
                  technicals={analysisData.technicals1d}
                  market={details?.market || 'VN'}
                  showSeasonality={false}
                />
              </div>
            </TabsContent>

            <TabsContent value="valuation" className="mt-6 space-y-6">
              <TickerValuationView
                technicals={analysisData.technicals1d}
                valuation={analysisData.valuation}
                symbol={details?.symbol}
              />
            </TabsContent>

            <TabsContent value="seasonality" className="mt-6">
              <SeasonalityStatsTable seasonality={analysisData.technicals1d.seasonality} />
            </TabsContent>
          </Tabs>
        </div>
      )}
    </div>
  );
}
