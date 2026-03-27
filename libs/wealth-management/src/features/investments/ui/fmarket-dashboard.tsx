'use client';

import { useFmarket } from '../model/use-fmarket';
import { GoldHistoryItem, UsdHistoryItem, GoldHistoryRaw, UsdHistoryRaw } from '../model/types';
import { Card, CardContent, CardHeader, CardTitle, CardDescription, Tabs, TabsContent, TabsList, TabsTrigger } from '@wealth-management/ui';
import { TrendingUp, Landmark, Award } from 'lucide-react';
import { useState, useEffect } from 'react';
import { FundTable, TickerDetails, BankRatesSection, GoldUsdSection } from '@wealth-management/features/investments/ui/fmarket';

export function FmarketDashboard() {
  const [mounted, setMounted] = useState(false);
  useEffect(() => {
    setMounted(true);
  }, []);

  const {
    stockFunds,
    bondFunds,
    balancedFunds,
    mmfFunds,
    bankRates,
    goldHistory,
    usdHistory,
    goldProducts,
    goldExtra,
    bankData,
    loading,
    goldLoading,
    usdLoading,
    error,
    goldRange,
    setGoldRange,
    usdRange,
    setUsdRange,
    selectedTicker,
    setSelectedTicker,
    tickerDetails,
    navHistory,
    detailsLoading,
    navRange,
    setNavRange,
  } = useFmarket();

  if (selectedTicker && tickerDetails) {
    return (
      <TickerDetails
        details={tickerDetails}
        history={navHistory}
        loading={detailsLoading}
        range={navRange}
        onRangeChange={setNavRange}
        onBack={() => setSelectedTicker(null)}
      />
    );
  }

  if (!mounted) return null;

  if (error) {
    return (
      <div className="p-4 border border-red-500/30 rounded-lg bg-red-500/10 text-red-400">
        <p className="font-bold">Error loading Fmarket data</p>
        <p className="text-sm">{error}</p>
      </div>
    );
  }

  const usdRate = goldExtra?.rateUsdToVnd || usdHistory[usdHistory.length - 1]?.rateSellUsd || 26350;
  const formattedGoldData: GoldHistoryItem[] = goldHistory
    .filter((item: GoldHistoryRaw) => item.reportDate)
    .map((item: GoldHistoryRaw) => {
      const timestamp = typeof item.reportDate === 'string' ? new Date(item.reportDate).getTime() : item.reportDate;
      return {
        date: new Date(timestamp).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }),
        vnAsk: item.askSjc / 1000000,
        vnBid: item.bidSjc / 1000000,
        world: (item.price * 1.2056 * usdRate) / 1000000,
        rawDate: timestamp,
      };
    })
    .sort((a, b) => a.rawDate - b.rawDate);

  const formattedUsdData: UsdHistoryItem[] = usdHistory
    .filter((item: UsdHistoryRaw) => item.reportDate)
    .map((item: UsdHistoryRaw) => {
      const timestamp = typeof item.reportDate === 'string' ? new Date(item.reportDate).getTime() : item.reportDate;
      return {
        date: new Date(timestamp).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }),
        price: item.rateSellUsd,
        rawDate: timestamp,
      };
    })
    .sort((a, b) => a.rawDate - b.rawDate);

  return (
    <div className="space-y-8">
      <Tabs defaultValue="stocks" className="w-full">
        <div className="flex items-center justify-between overflow-x-auto pb-2 scrollbar-hide">
          <TabsList className="bg-muted/50 p-1 rounded-xl border border-border/50">
            <TabsTrigger
              value="stocks"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Cổ phiếu
            </TabsTrigger>
            <TabsTrigger
              value="bonds"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Trái phiếu
            </TabsTrigger>
            <TabsTrigger
              value="balanced"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Cân bằng
            </TabsTrigger>
            <TabsTrigger
              value="mmf"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Tiền tệ
            </TabsTrigger>
            <TabsTrigger
              value="banks"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Ngân hàng
            </TabsTrigger>
            <TabsTrigger
              value="gold"
              className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm"
            >
              Vàng & USD
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent
          value="stocks"
          className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300"
        >
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-primary/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-primary">
                <TrendingUp className="h-5 w-5 text-emerald-500" /> Lợi nhuận Quỹ Cổ phiếu
              </CardTitle>
              <CardDescription>
                Bảng xếp hạng hiệu quả đầu tư các quỹ mở cổ phiếu uy tín thành viên Fmarket
              </CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={stockFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent
          value="bonds"
          className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300"
        >
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-indigo-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-indigo-700">
                <Award className="h-5 w-5 text-indigo-500" /> Lợi nhuận Quỹ Trái phiếu
              </CardTitle>
              <CardDescription>Lựa chọn tối ưu cho mục tiêu bảo toàn vốn và lợi nhuận ổn định</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={bondFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent
          value="balanced"
          className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300"
        >
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-emerald-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-emerald-700">
                <TrendingUp className="h-5 w-5 text-emerald-500" /> Lợi nhuận Quỹ Cân bằng
              </CardTitle>
              <CardDescription>Kết hợp linh hoạt giữa cổ phiếu và trái phiếu để tối ưu rủi ro</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={balancedFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="mmf" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-blue-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-blue-700">
                <Landmark className="h-5 w-5 text-blue-500" /> Lợi nhuận Quỹ Tiền tệ (MMF)
              </CardTitle>
              <CardDescription>An toàn cao, thanh khoản linh hoạt tương đương tiền gửi tiết kiệm</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={mmfFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="banks" className="mt-6">
          <BankRatesSection loading={loading} bankData={bankData} bankRates={bankRates} />
        </TabsContent>

        <TabsContent value="gold" className="mt-6">
          <GoldUsdSection
            loading={loading}
            goldLoading={goldLoading}
            usdLoading={usdLoading}
            goldRange={goldRange}
            setGoldRange={setGoldRange}
            usdRange={usdRange}
            setUsdRange={setUsdRange}
            formattedGoldData={formattedGoldData}
            formattedUsdData={formattedUsdData}
            goldProducts={goldProducts}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}
