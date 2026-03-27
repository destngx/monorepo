'use client';

import { useFmarket } from '../model/use-fmarket';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@wealth-management/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp, Landmark, Award, RefreshCcw } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend, BarChart, Bar } from 'recharts';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useState, useEffect } from 'react';

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
    issuers,
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
    setNavRange
  } = useFmarket();

  if (selectedTicker && tickerDetails) {
    return <TickerDetails 
      details={tickerDetails} 
      history={navHistory} 
      loading={detailsLoading} 
      range={navRange}
      onRangeChange={setNavRange}
      onBack={() => setSelectedTicker(null)} 
    />;
  }

  if (!mounted) return null;

  if (error) {
    return (
      <div className="p-4 border border-red-500/30 rounded-lg bg-red-500/10 text-red-400">
        <p className="font-bold">Error loading Fmarket data</p>
        <p className="text-sm">{error.message}</p>
      </div>
    );
  }

  // Calculate the world gold price in VND/Lượng for comparison
  const usdRate = goldExtra?.rateUsdToVnd || usdHistory[usdHistory.length - 1]?.rateSellUsd || 26350;
  const formattedGoldData = goldHistory
    .filter((item: any) => item.reportDate)
    .map((item: any) => {
      const timestamp = typeof item.reportDate === 'string' ? new Date(item.reportDate).getTime() : item.reportDate;
      return {
        date: new Date(timestamp).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }),
        vnAsk: item.askSjc / 1000000,
        vnBid: item.bidSjc / 1000000,
        world: (item.price * 1.2056 * usdRate) / 1000000,
        rawDate: timestamp
      };
    })
    .sort((a: any, b: any) => a.rawDate - b.rawDate);

  const formattedUsdData = usdHistory
    .filter((item: any) => item.reportDate)
    .map((item: any) => {
      const timestamp = typeof item.reportDate === 'string' ? new Date(item.reportDate).getTime() : item.reportDate;
      return {
        date: new Date(timestamp).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }),
        price: item.rateSellUsd,
        rawDate: timestamp
      };
    })
    .sort((a: any, b: any) => a.rawDate - b.rawDate);

  const latestGoldPrice = goldProducts[0]?.price || (goldHistory[goldHistory.length - 1]?.askSjc / 10);

  return (
    <div className="space-y-8">
      <Tabs defaultValue="stocks" className="w-full">
        <div className="flex items-center justify-between overflow-x-auto pb-2 scrollbar-hide">
          <TabsList className="bg-muted/50 p-1 rounded-xl border border-border/50">
            <TabsTrigger value="stocks" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Cổ phiếu
            </TabsTrigger>
            <TabsTrigger value="bonds" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Trái phiếu
            </TabsTrigger>
            <TabsTrigger value="balanced" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Cân bằng
            </TabsTrigger>
            <TabsTrigger value="mmf" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Tiền tệ
            </TabsTrigger>
            <TabsTrigger value="banks" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Ngân hàng
            </TabsTrigger>
            <TabsTrigger value="gold" className="rounded-lg px-6 py-2 transition-all data-[state=active]:bg-white data-[state=active]:text-primary data-[state=active]:shadow-sm">
              Vàng & USD
            </TabsTrigger>
          </TabsList>
        </div>

        {/* Cổ phiếu Tab Content */}
        <TabsContent value="stocks" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-primary/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-primary">
                <TrendingUp className="h-5 w-5 text-emerald-500" />
                Lợi nhuận Quỹ Cổ phiếu
              </CardTitle>
              <CardDescription>Bảng xếp hạng hiệu quả đầu tư các quỹ mở cổ phiếu uy tín thành viên Fmarket</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={stockFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Trái phiếu Tab Content */}
        <TabsContent value="bonds" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-indigo-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-indigo-700">
                < Award className="h-5 w-5 text-indigo-500" />
                Lợi nhuận Quỹ Trái phiếu
              </CardTitle>
              <CardDescription>Lựa chọn tối ưu cho mục tiêu bảo toàn vốn và lợi nhuận ổn định</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={bondFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Cân bằng Tab Content */}
        <TabsContent value="balanced" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-emerald-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-emerald-700">
                <TrendingUp className="h-5 w-5 text-emerald-500" />
                Lợi nhuận Quỹ Cân bằng
              </CardTitle>
              <CardDescription>Kết hợp linh hoạt giữa cổ phiếu và trái phiếu để tối ưu rủi ro</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={balancedFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Tiền tệ Tab Content */}
        <TabsContent value="mmf" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-blue-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-blue-700">
                <Landmark className="h-5 w-5 text-blue-500" />
                Lợi nhuận Quỹ Tiền tệ (MMF)
              </CardTitle>
              <CardDescription>An toàn cao, thanh khoản linh hoạt tương đương tiền gửi tiết kiệm</CardDescription>
            </CardHeader>
            <CardContent className="p-0">
              <FundTable funds={mmfFunds} loading={loading} onSelect={setSelectedTicker} />
            </CardContent>
          </Card>
        </TabsContent>

        {/* Ngân hàng Tab Content */}
        <TabsContent value="banks" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <Card className="shadow-sm border-border/50">
            <CardHeader className="bg-amber-500/5 border-b border-border/50">
              <CardTitle className="text-lg font-bold flex items-center gap-2 text-amber-700">
                <Landmark className="h-5 w-5 text-amber-500" />
                Lãi suất huy động Ngân hàng
              </CardTitle>
              <CardDescription>Bảng so sánh lãi suất gửi tiết kiệm mới nhất tại các ngân hàng lớn</CardDescription>
            </CardHeader>
            <CardContent className="pt-6">
              {loading ? (
                <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-4">
                  {[...Array(12)].map((_, i) => (
                    <Skeleton key={i} className="h-20 w-full rounded-xl" />
                  ))}
                </div>
              ) : (
                <div className="space-y-6">
                  {bankData?.note && (
                    <div className="p-4 rounded-xl bg-amber-500/5 border border-amber-500/10 text-xs text-amber-900 leading-relaxed italic">
                      " {bankData.note} "
                    </div>
                  )}

                  <div className="h-[300px] w-full mt-4">
                    <ResponsiveContainer width="100%" height="100%">
                      <BarChart data={bankRates.map((rate: any) => ({
                        name: rate.name || rate.bankShortName || 'Bank',
                        rate: rate.value || rate.interestRate
                      }))} margin={{ top: 20, right: 30, left: 0, bottom: 60 }}>
                        <defs>
                          <linearGradient id="barGradient" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#f59e0b" stopOpacity={0.8}/>
                            <stop offset="95%" stopColor="#d97706" stopOpacity={0.3}/>
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(0,0,0,0.05)" />
                        <XAxis 
                          dataKey="name" 
                          angle={-45} 
                          textAnchor="end" 
                          interval={0} 
                          height={100} 
                          tick={{ fontSize: 10, fontWeight: 700 }}
                          axisLine={false}
                          tickLine={false}
                        />
                        <YAxis 
                          tick={{ fontSize: 10, fontWeight: 600 }}
                          axisLine={false}
                          tickLine={false}
                          tickFormatter={(val) => `${val}%`}
                        />
                        <Tooltip 
                          cursor={{ fill: 'rgba(245, 158, 11, 0.05)' }}
                          content={({ active, payload, label }) => {
                            if (active && payload && payload.length) {
                              return (
                                <div className="bg-white p-3 border border-border/50 shadow-2xl rounded-xl">
                                  <p className="text-[10px] font-black uppercase tracking-wider text-muted-foreground mb-1">{label}</p>
                                  <p className="text-sm font-black text-amber-600">{payload[0].value}% <span className="text-[10px] lowercase font-medium text-muted-foreground">/ năm</span></p>
                                </div>
                              );
                            }
                            return null;
                          }}
                        />
                        <Bar 
                          dataKey="rate" 
                          fill="url(#barGradient)" 
                          radius={[6, 6, 0, 0]}
                          barSize={32}
                        />
                      </BarChart>
                    </ResponsiveContainer>
                  </div>

                  <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3 mt-6">
                    {bankRates.map((rate: any, i: number) => (
                      <div 
                        key={i} 
                        className="p-3 rounded-xl border border-border/50 bg-muted/20 hover:bg-muted/30 transition-all flex flex-col justify-between group"
                      >
                        <div className="text-[10px] uppercase font-bold text-muted-foreground truncate group-hover:text-amber-700">
                          {rate.name || rate.bankShortName || 'Bank'}
                        </div>
                        <div className="flex items-baseline justify-between mt-1">
                          <span className="text-lg font-black text-amber-600">
                            {rate.value || rate.interestRate}%
                          </span>
                          <span className="text-[9px] text-muted-foreground font-medium lowercase">
                            năm
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>

                  {(bankData?.updateAt || bankData?.sourceNote) && (
                    <div className="text-[10px] text-muted-foreground text-right italic">
                      Cập nhật lúc {bankData.updateAt} {bankData.sourceNote ? `theo ${bankData.sourceNote}` : ''}
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Vàng Tab Content */}
        <TabsContent value="gold" className="space-y-6 mt-6 focus-visible:outline-none animate-in fade-in duration-300">
          <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
            <div className="lg:col-span-1 space-y-6">
              {/* Gold Price Card */}
              {/* Gold Prices Card */}
              <Card className="shadow-sm border-border/50 overflow-hidden bg-gradient-to-br from-amber-500/5 to-transparent">
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-black uppercase tracking-widest text-amber-600 flex items-center gap-2">
                    <TrendingUp className="h-4 w-4" /> Bảng giá Vàng trực tuyến
                  </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-1">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Vàng miếng SJC</div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-[9px] font-bold text-muted-foreground">Mua vào</div>
                        <div className="text-lg font-black text-amber-600">172,000,000</div>
                      </div>
                      <div>
                        <div className="text-[9px] font-bold text-muted-foreground">Bán ra</div>
                        <div className="text-lg font-black text-red-600">175,000,000</div>
                      </div>
                    </div>
                  </div>

                  <div className="space-y-1">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Vàng nhẫn 99.99</div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <div className="text-[9px] font-bold text-muted-foreground">Mua vào</div>
                        <div className="text-lg font-black text-amber-600">172,000,000</div>
                      </div>
                      <div>
                        <div className="text-[9px] font-bold text-muted-foreground">Bán ra</div>
                        <div className="text-lg font-black text-red-600">175,000,000</div>
                      </div>
                    </div>
                  </div>

                  <div className="pt-2 border-t border-border/50">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Vàng thế giới</div>
                    <div className="flex items-baseline gap-2">
                      <span className="text-lg font-black text-blue-600">145,093,896</span>
                      <span className="text-[10px] font-bold text-muted-foreground">VND</span>
                    </div>
                  </div>

                  <div className="pt-2 border-t border-border/50">
                    <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Tỷ giá USD/VND</div>
                    <div className="flex items-baseline gap-2">
                      <span className="text-lg font-black text-blue-700">26,359</span>
                      <span className="text-[10px] font-bold text-muted-foreground">VND</span>
                    </div>
                  </div>

                  <div className="text-[10px] text-right font-medium text-muted-foreground mt-4 italic">
                    Cập nhật Ngày 25/03/2026
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Gold Trend Chart */}
            <Card className="lg:col-span-3 shadow-sm border-border/50">
              <CardHeader className="pb-2">
                <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                  <div>
                    <CardTitle className="text-sm font-bold flex items-center gap-2 uppercase tracking-tight">
                      <TrendingUp className="h-4 w-4 text-emerald-500" /> Biểu đồ Giá Vàng Việt Nam & Thế Giới
                    </CardTitle>
                    <CardDescription className="text-[10px]">So sánh tương quan giá vàng SJC và giá thế giới quy đổi (Triệu VND/Lượng)</CardDescription>
                  </div>
                  <div className="flex bg-muted/50 p-0.5 rounded-lg border border-border/50 self-start">
                    {(['YTD', '6M', '1Y', '3Y', '5Y', 'ALL'] as const).map((r) => (
                      <button
                        key={r}
                        onClick={() => setGoldRange(r)}
                        className={`px-2 py-1 text-[9px] font-bold rounded-md transition-all ${
                          goldRange === r 
                            ? 'bg-white text-primary shadow-sm' 
                            : 'text-muted-foreground hover:text-foreground'
                        }`}
                      >
                        {r}
                      </button>
                    ))}
                  </div>
                </div>
              </CardHeader>
              <CardContent className="h-[220px] min-h-[220px] pt-4 pr-0 relative">
                {(loading || goldLoading) ? (
                  <Skeleton className="h-full w-full" />
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={formattedGoldData}>
                      <defs>
                        <linearGradient id="colorVnAsk" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#ef4444" stopOpacity={0.2}/>
                          <stop offset="95%" stopColor="#ef4444" stopOpacity={0}/>
                        </linearGradient>
                        <linearGradient id="colorVnBid" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#10b981" stopOpacity={0.2}/>
                          <stop offset="95%" stopColor="#10b981" stopOpacity={0}/>
                        </linearGradient>
                        <linearGradient id="colorWorld" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.1}/>
                          <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                        </linearGradient>
                      </defs>
                      <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(0,0,0,0.05)" />
                      <XAxis 
                        dataKey="date" 
                        fontSize={9} 
                        tickLine={false} 
                        axisLine={false}
                        minTickGap={40}
                      />
                      <YAxis 
                        fontSize={9} 
                        tickLine={false} 
                        axisLine={false} 
                        domain={['auto', 'auto']}
                        tickFormatter={(val) => `${val}M`}
                      />
                      <Tooltip 
                        contentStyle={{ fontSize: '11px', borderRadius: '12px', border: 'none', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }}
                        labelStyle={{ fontWeight: 'black', marginBottom: '8px', color: '#111' }}
                        cursor={{ stroke: '#ccc', strokeWidth: 1, strokeDasharray: '4 4' }}
                      />
                      <Legend verticalAlign="top" height={30} iconType="circle" wrapperStyle={{ fontSize: '10px', paddingTop: '0' }} />
                      <Area 
                        type="monotone" 
                        dataKey="vnAsk" 
                        name="SJC Bán ra (VN)" 
                        stroke="#ef4444" 
                        strokeWidth={2}
                        fillOpacity={1} 
                        fill="url(#colorVnAsk)" 
                        activeDot={{ r: 4, strokeWidth: 0 }}
                      />
                      <Area 
                        type="monotone" 
                        dataKey="vnBid" 
                        name="SJC Mua vào (VN)" 
                        stroke="#10b981" 
                        strokeWidth={2}
                        fillOpacity={1} 
                        fill="url(#colorVnBid)" 
                        activeDot={{ r: 4, strokeWidth: 0 }}
                      />
                      <Area 
                        type="monotone" 
                        dataKey="world" 
                        name="Thế giới (Quy đổi)" 
                        stroke="#3b82f6" 
                        strokeWidth={1.5}
                        strokeDasharray="4 4"
                        fillOpacity={1} 
                        fill="url(#colorWorld)" 
                      activeDot={{ r: 3, strokeWidth: 0 }}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                )}
              </CardContent>
            </Card>
          </div>

          {/* USD Trend Chart */}
          <Card className="shadow-sm border-border/50">
            <CardHeader className="pb-2">
              <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                <div>
                  <CardTitle className="text-sm font-bold flex items-center gap-2 uppercase tracking-tight text-blue-700">
                    <Landmark className="h-4 w-4 text-blue-500" /> Biểu đồ Tỷ giá USD/VND (VCB)
                  </CardTitle>
                  <CardDescription className="text-xs">Lịch sử tỷ giá bán ra tại Vietcombank trong tháng qua</CardDescription>
                </div>
                <div className="flex bg-muted/50 p-0.5 rounded-lg border border-border/50 self-start">
                  {(['YTD', '6M', '1Y', '3Y', '5Y', 'ALL'] as const).map((r) => (
                    <button
                      key={r}
                      onClick={() => setUsdRange(r)}
                      className={`px-2 py-1 text-[9px] font-bold rounded-md transition-all ${
                        usdRange === r 
                          ? 'bg-white text-primary shadow-sm' 
                          : 'text-muted-foreground hover:text-foreground'
                      }`}
                    >
                      {r}
                    </button>
                  ))}
                </div>
              </div>
            </CardHeader>
            <CardContent className="h-[220px] min-h-[220px] pt-4 pr-0 relative">
              {(loading || usdLoading) ? (
                <Skeleton className="h-full w-full" />
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={formattedUsdData}>
                    <defs>
                      <linearGradient id="colorUsd" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2}/>
                        <stop offset="95%" stopColor="#3b82f6" stopOpacity={0}/>
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(0,0,0,0.05)" />
                    <XAxis 
                      dataKey="date" 
                      fontSize={9} 
                      tickLine={false} 
                      axisLine={false}
                      minTickGap={40}
                    />
                    <YAxis 
                      fontSize={9} 
                      tickLine={false} 
                      axisLine={false} 
                      domain={['auto', 'auto']}
                      tickFormatter={(val) => Math.round(val).toLocaleString()}
                    />
                    <Tooltip 
                      contentStyle={{ fontSize: '11px', borderRadius: '12px', border: 'none', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }}
                      labelStyle={{ fontWeight: 'black', marginBottom: '8px', color: '#111' }}
                      cursor={{ stroke: '#ccc', strokeWidth: 1, strokeDasharray: '4 4' }}
                    />
                    <Area 
                      type="monotone" 
                      dataKey="price" 
                      name="USD/VND" 
                      stroke="#3b82f6" 
                      strokeWidth={2}
                      fillOpacity={1} 
                      fill="url(#colorUsd)" 
                      activeDot={{ r: 4, strokeWidth: 0 }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>

          <Card className="shadow-sm border-border/50 overflow-hidden">
            <CardHeader className="pb-4 bg-amber-500/5 border-b border-border/50">
              <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
                <div>
                  <CardTitle className="text-lg font-bold flex items-center gap-2 text-amber-700">
                    < Award className="h-5 w-5 text-amber-500" />
                    Bảng giá Vàng trực tuyến
                  </CardTitle>
                  <CardDescription className="text-xs">
                    Thời gian giao dịch vàng từ 9h00 - 18h30 | Cập nhật lúc {new Date().toLocaleTimeString('vi-VN', {hour: '2-digit', minute:'2-digit'})}, {new Date().toLocaleDateString('vi-VN')}
                  </CardDescription>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
              <GoldProductTable products={goldProducts} loading={loading} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function GoldProductTable({ products, loading }: { products: any[], loading: boolean }) {
  return (
    <Table>
      <TableHeader className="bg-muted/30">
        <TableRow>
          <TableHead className="w-[300px] text-[11px] uppercase font-bold text-muted-foreground">Loại vàng</TableHead>
          <TableHead className="text-[11px] uppercase font-bold text-muted-foreground">Công ty</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Mua vào (VND/chỉ)</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Bán ra (VND/chỉ)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <TableRow key={i}>
              <TableCell><Skeleton className="h-10 w-full" /></TableCell>
              <TableCell><Skeleton className="h-4 w-20" /></TableCell>
              <TableCell><Skeleton className="h-4 w-24 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-24 ml-auto" /></TableCell>
            </TableRow>
          ))
        ) : (
          products.map((product) => {
            const isPaused = product.status !== 'PRODUCT_ACTIVE' || 
                            product.productGold?.isDisable === true || 
                            !product.productGold?.buyPrice;
            return (
              <TableRow key={product.id} className="hover:bg-muted/50 transition-colors group">
                <TableCell>
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-8 rounded bg-amber-100 flex items-center justify-center overflow-hidden border border-amber-200">
                      {product.owner?.avatarUrl ? (
                        <img src={product.owner.avatarUrl} alt={product.shortName} className="h-full w-full object-cover" />
                      ) : (
                        <TrendingUp className="h-4 w-4 text-amber-600" />
                      )}
                    </div>
                    <div>
                      <div className="font-bold text-sm text-amber-900">{product.name}</div>
                      <div className="text-[10px] text-muted-foreground">{product.shortName}</div>
                    </div>
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-[11px] font-bold text-muted-foreground">{product.owner?.shortName}</span>
                </TableCell>
                <TableCell className="text-right font-mono">
                  {isPaused ? (
                    <span className="text-muted-foreground/50">-</span>
                  ) : (
                    <span className="font-bold text-sm text-emerald-600">
                      {product.productGold?.buyPrice?.toLocaleString('vi-VN')}
                    </span>
                  )}
                </TableCell>
                <TableCell className="text-right font-mono">
                  {isPaused ? (
                    <span className="text-[10px] font-bold text-red-500 bg-red-50 px-2 py-0.5 rounded">Tạm ngừng bán</span>
                  ) : (
                    <span className="font-bold text-sm text-red-600">
                      {product.productGold?.sellPrice?.toLocaleString('vi-VN')}
                    </span>
                  )}
                </TableCell>
              </TableRow>
            );
          })
        )}
      </TableBody>
    </Table>
  );
}

function FundTable({ funds, loading, onSelect }: { funds: any[], loading: boolean, onSelect?: (ticker: string) => void }) {
  const mapAssetType = (type: string) => {
    switch (type) {
      case 'STOCK_FUND': return 'Quỹ cổ phiếu';
      case 'BOND_FUND': return 'Quỹ trái phiếu';
      case 'BALANCED_FUND': return 'Quỹ cân bằng';
      default: return 'Quỹ mở';
    }
  };

  return (
    <Table>
      <TableHeader className="bg-muted/30">
        <TableRow>
          <TableHead className="w-[200px] text-[11px] uppercase font-bold text-muted-foreground">Tên CCQ/Loại quỹ</TableHead>
          <TableHead className="text-[11px] uppercase font-bold text-muted-foreground">Tổ chức phát hành</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Giá gần nhất</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">Lợi nhuận 1 năm gần nhất</TableHead>
          <TableHead className="text-right text-[11px] uppercase font-bold text-muted-foreground">LN bình quân (3 năm)</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading ? (
          Array.from({ length: 5 }).map((_, i) => (
            <TableRow key={i}>
              <TableCell><Skeleton className="h-10 w-full" /></TableCell>
              <TableCell><Skeleton className="h-4 w-20" /></TableCell>
              <TableCell><Skeleton className="h-8 w-24 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-12 ml-auto" /></TableCell>
              <TableCell><Skeleton className="h-4 w-12 ml-auto" /></TableCell>
            </TableRow>
          ))
        ) : (
          funds.map((fund) => (
            <TableRow 
              key={fund.id} 
              className="hover:bg-muted/50 transition-colors group cursor-pointer"
              onClick={() => onSelect?.(fund.shortName || fund.code)}
            >
              <TableCell>
                <div className="flex items-center gap-3">
                  <div className="h-8 w-8 rounded-full bg-primary/5 flex items-center justify-center overflow-hidden border border-border/50 group-hover:border-primary/30 transition-colors">
                    {fund.owner?.avatarUrl ? (
                      <img src={fund.owner.avatarUrl} alt={fund.code} className="h-full w-full object-cover shadow-sm" />
                    ) : (
                      <div className="text-[10px] font-bold text-primary">{fund.code.slice(0, 3)}</div>
                    )}
                  </div>
                  <div>
                    <div className="font-bold text-sm text-primary group-hover:text-primary/80">{fund.code}</div>
                    <div className="text-[9px] text-muted-foreground whitespace-nowrap">{mapAssetType(fund.fundAssetType)}</div>
                  </div>
                </div>
              </TableCell>
              <TableCell>
                <span className="text-[11px] font-semibold text-muted-foreground/80">{fund.owner?.shortName}</span>
              </TableCell>
              <TableCell className="text-right font-mono">
                <div className="font-bold text-sm">{fund.nav?.toLocaleString('vi-VN')}</div>
                <div className="text-[9px] text-muted-foreground leading-none mt-1">
                  Theo NAV tại {fund.productNavChange?.updateAt ? new Date(fund.productNavChange.updateAt).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' }) : '--/--'}
                </div>
              </TableCell>
              <TableCell className="text-right">
                <span className="text-emerald-500 font-black text-sm">
                  {fund.productNavChange?.navTo12Months?.toFixed(2)}%
                </span>
              </TableCell>
              <TableCell className="text-right">
                <span className="text-indigo-500 font-black text-sm">
                  {fund.productNavChange?.annualizedReturn36Months?.toFixed(2)}%
                </span>
              </TableCell>
            </TableRow>
          ))
        )}
      </TableBody>
    </Table>
  );
}
function TickerDetails({ 
  details, 
  history, 
  loading, 
  range,
  onRangeChange,
  onBack 
}: { 
  details: any, 
  history: any[], 
  loading: boolean, 
  range: any,
  onRangeChange: (r: any) => void,
  onBack: () => void 
}) {
  if (loading && !history.length) return <div className="p-10 text-center flex flex-col items-center gap-4"><Skeleton className="h-40 w-full rounded-2xl" /><Skeleton className="h-80 w-full rounded-2xl" /></div>;

  const formattedHistory = history.map(item => {
    const timestamp = typeof item.createdAt === 'string' ? new Date(item.createdAt).getTime() : item.createdAt;
    return {
      date: new Date(timestamp).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: '2-digit' }),
      nav: item.nav,
      rawDate: timestamp
    };
  }).sort((a, b) => a.rawDate - b.rawDate);

  return (
    <div className="space-y-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <button 
        onClick={onBack}
        className="flex items-center gap-2 text-sm font-bold text-muted-foreground hover:text-primary transition-colors mb-4 group"
      >
        <TrendingUp className="h-4 w-4 rotate-180 group-hover:-translate-x-1 transition-transform" />
        Quay lại danh sách
      </button>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        <div className="lg:col-span-1 space-y-6">
          <Card className="shadow-xl border-border/50 overflow-hidden h-fit sticky top-6">
             <CardHeader className="bg-gradient-to-r from-primary/5 to-primary/10 border-b border-border/50 p-6">
              <div className="flex flex-col items-center text-center gap-4">
                <div className="h-20 w-20 rounded-2xl bg-white shadow-sm border border-border/50 flex items-center justify-center overflow-hidden p-2">
                  <img src={details.owner?.avatarUrl} alt={details.shortName} className="max-h-full max-w-full object-contain" />
                </div>
                <div>
                  <h1 className="text-lg font-black text-primary leading-tight">{details.name}</h1>
                  <div className="flex items-center justify-center gap-2 mt-2">
                    <span className="px-2 py-0.5 bg-primary text-white text-[10px] font-black rounded uppercase">{details.shortName}</span>
                    <span className="text-[10px] font-bold text-muted-foreground uppercase opacity-70 tracking-wider">{details.code}</span>
                  </div>
                </div>
              </div>
            </CardHeader>
            <CardContent className="p-0">
               <div className="divide-y divide-border/50">
                  <div className="p-4 flex items-center justify-between">
                    <span className="text-[10px] font-bold text-muted-foreground uppercase">Tổ chức</span>
                    <span className="text-xs font-black text-primary uppercase text-right">{details.owner?.shortName}</span>
                  </div>
                  <div className="p-4 flex items-center justify-between">
                    <span className="text-[10px] font-bold text-muted-foreground uppercase">Loại quỹ</span>
                    <span className="text-xs font-black text-primary uppercase text-right">{details.fundAssetType === 'STOCK_FUND' ? 'Cổ phiếu' : 'Trái phiếu'}</span>
                  </div>
                  <div className="p-4 flex items-center justify-between">
                    <span className="text-[10px] font-bold text-muted-foreground uppercase">Rủi ro</span>
                    <span className="text-xs font-black text-amber-600 uppercase text-right">{details.riskLevel?.name}</span>
                  </div>
               </div>
            </CardContent>
          </Card>
        </div>

        <div className="lg:col-span-3">
          <Tabs defaultValue="overview" className="w-full space-y-6">
            <TabsList className="bg-muted/50 p-1 rounded-xl border border-border/50 w-full justify-start overflow-x-auto no-scrollbar">
              <TabsTrigger value="overview" className="px-6 data-[state=active]:bg-white data-[state=active]:shadow-sm transition-all rounded-lg">Tổng quan</TabsTrigger>
              <TabsTrigger value="performance" className="px-6 data-[state=active]:bg-white data-[state=active]:shadow-sm transition-all rounded-lg">Biểu đồ & Hiệu quả</TabsTrigger>
              <TabsTrigger value="portfolio" className="px-6 data-[state=active]:bg-white data-[state=active]:shadow-sm transition-all rounded-lg">Danh mục</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-6 animate-in fade-in duration-300 focus-visible:outline-none">
               <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                 <Card className="shadow-lg border-border/50 overflow-hidden">
                    <CardHeader className="pb-4">
                      <CardTitle className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Chỉ số cơ bản</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-6 pt-0">
                       <div className="flex items-end justify-between">
                         <div>
                           <div className="text-[9px] font-bold text-muted-foreground uppercase mb-1">Giá gần nhất (NAV)</div>
                           <div className="text-3xl font-black text-primary">{details.nav?.toLocaleString('vi-VN')}</div>
                         </div>
                         <div className={`text-sm font-black flex items-center gap-1 mb-1 ${details.productNavChange?.navToPrevious >= 0 ? 'text-emerald-500' : 'text-red-500'}`}>
                            {details.productNavChange?.navToPrevious >= 0 ? '▲' : '▼'} {Math.abs(details.productNavChange?.navToPrevious)}%
                         </div>
                       </div>
                       
                       <div className="grid grid-cols-2 gap-4">
                          <div className="p-3 bg-emerald-500/5 rounded-xl border border-emerald-500/10">
                             <div className="text-[9px] font-bold text-emerald-600 uppercase">LN 12 Tháng</div>
                             <div className="text-xl font-black text-emerald-600">{details.productNavChange?.navTo12Months}%</div>
                          </div>
                          <div className="p-3 bg-indigo-500/5 rounded-xl border border-indigo-500/10">
                             <div className="text-[9px] font-bold text-indigo-600 uppercase">Mua tối thiểu</div>
                             <div className="text-xl font-black text-indigo-600">{details.buyMinValue?.toLocaleString('vi-VN')} đ</div>
                          </div>
                       </div>
                    </CardContent>
                 </Card>

                 <Card className="shadow-lg border-border/50 overflow-hidden">
                    <CardHeader className="pb-4">
                      <CardTitle className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Giao dịch tiếp theo</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4 pt-0">
                        <div className="p-4 bg-primary/5 rounded-2xl border border-primary/10 flex items-center justify-between">
                          <div>
                            <div className="text-[9px] font-bold text-muted-foreground uppercase">Phiên khớp lệnh</div>
                            <div className="text-lg font-black text-primary">27/03/2026</div>
                          </div>
                          <div className="text-right">
                            <div className="text-[9px] font-bold text-muted-foreground uppercase">Hạn đặt lệnh</div>
                            <div className="text-sm font-black text-amber-600">11:00 am, 26/03</div>
                          </div>
                        </div>
                        <div className="flex justify-between items-center px-2">
                           <span className="text-[10px] font-bold text-muted-foreground uppercase tracking-wider">Tài sản ròng</span>
                           <span className="text-sm font-black text-primary">{(details.totalNav / 1000000000).toFixed(1)} Tỷ VND</span>
                        </div>
                    </CardContent>
                 </Card>
               </div>

               <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <Card className="shadow-lg border-border/50 overflow-hidden">
                     <CardHeader className="pb-2">
                       <CardTitle className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Chi phí giao dịch</CardTitle>
                     </CardHeader>
                     <CardContent className="space-y-4">
                        <div className="space-y-2">
                           <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Phí mua</div>
                           <div className="grid grid-cols-2 gap-2 text-center">
                              <div className="p-2 border border-border/50 rounded-lg bg-muted/20">
                                <div className="text-[9px] font-bold text-muted-foreground whitespace-nowrap">&le; 20 tỷ</div>
                                <div className="text-sm font-black text-emerald-600">0.75%</div>
                              </div>
                              <div className="p-2 border border-border/50 rounded-lg bg-muted/20">
                                <div className="text-[9px] font-bold text-muted-foreground whitespace-nowrap">&gt; 20 tỷ</div>
                                <div className="text-sm font-black text-emerald-600">0.75%</div>
                              </div>
                           </div>
                        </div>
                        <div className="space-y-2">
                           <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Phí bán</div>
                           <div className="grid grid-cols-2 gap-2 text-center">
                              <div className="p-2 border border-border/50 rounded-lg bg-muted/20">
                                <div className="text-[9px] font-bold text-muted-foreground whitespace-nowrap">&le; 12 tháng</div>
                                <div className="text-sm font-black text-red-600">1.25%</div>
                              </div>
                              <div className="p-2 border border-border/50 rounded-lg bg-muted/20">
                                <div className="text-[9px] font-bold text-muted-foreground whitespace-nowrap">&gt; 12 tháng</div>
                                <div className="text-sm font-black text-emerald-600">0%</div>
                              </div>
                           </div>
                        </div>
                     </CardContent>
                  </Card>

                  <Card className="shadow-lg border-border/50 overflow-hidden">
                    <CardHeader className="pb-2">
                      <CardTitle className="text-[10px] font-black uppercase tracking-widest text-muted-foreground">Mô tả quỹ</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <p className="text-sm leading-relaxed text-muted-foreground font-medium italic mb-4">
                        "{details.description}"
                      </p>
                      {details.contentHome?.content && (
                         <div className="text-xs text-muted-foreground/80 leading-relaxed prose-sm line-clamp-6 opacity-80" dangerouslySetInnerHTML={{ __html: details.contentHome.content }} />
                      )}
                    </CardContent>
                  </Card>
               </div>
            </TabsContent>

            <TabsContent value="performance" className="space-y-6 animate-in fade-in duration-300 focus-visible:outline-none">
               <Card className="shadow-xl border-border/50 overflow-hidden">
                <CardHeader className="pb-2">
                  <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
                    <CardTitle className="text-sm font-black uppercase tracking-widest text-primary flex items-center gap-2">
                       <TrendingUp className="h-4 w-4 text-emerald-500" /> Biểu đồ NAV Lịch sử
                    </CardTitle>
                    <div className="flex bg-muted/50 p-0.5 rounded-lg border border-border/50 self-start">
                      {(['1M', '6M', 'YTD', '1Y', '2Y', '3Y', '5Y', 'ALL'] as const).map((r) => (
                        <button
                          key={r}
                          onClick={() => onRangeChange(r)}
                          className={`px-3 py-1 text-[10px] font-black rounded-md transition-all ${
                            range === r 
                              ? 'bg-white text-primary shadow-sm' 
                              : 'text-muted-foreground hover:text-foreground'
                          }`}
                        >
                          {r}
                        </button>
                      ))}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="h-[400px] min-h-[400px] pt-6 pr-0 relative">
                   {loading && <div className="absolute inset-0 z-10 bg-white/50 flex items-center justify-center backdrop-blur-[1px]"><RefreshCcw className="h-8 w-8 animate-spin text-primary opacity-20" /></div>}
                   <ResponsiveContainer width="100%" height="100%">
                      <AreaChart data={formattedHistory}>
                        <defs>
                          <linearGradient id="colorNav" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="#1e3a8a" stopOpacity={0.2}/>
                            <stop offset="95%" stopColor="#1e3a8a" stopOpacity={0}/>
                          </linearGradient>
                        </defs>
                        <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="rgba(0,0,0,0.05)" />
                        <XAxis dataKey="date" fontSize={9} tickLine={false} axisLine={false} minTickGap={50} />
                        <YAxis fontSize={9} tickLine={false} axisLine={false} domain={['auto', 'auto']} tickFormatter={(v) => v.toLocaleString()} />
                        <Tooltip 
                          contentStyle={{ fontSize: '11px', borderRadius: '12px', border: 'none', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }}
                          labelStyle={{ fontWeight: 'black', marginBottom: '8px', color: '#111' }}
                          cursor={{ stroke: '#ccc', strokeWidth: 1, strokeDasharray: '4 4' }}
                        />
                        <Area type="monotone" dataKey="nav" stroke="#1e3a8a" strokeWidth={3} fillOpacity={1} fill="url(#colorNav)" />
                      </AreaChart>
                   </ResponsiveContainer>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="portfolio" className="space-y-6 animate-in fade-in duration-300 focus-visible:outline-none">
               <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  <Card className="shadow-xl border-border/50 overflow-hidden">
                     <CardHeader className="bg-muted/50 border-b border-border/50">
                       <CardTitle className="text-sm font-black uppercase tracking-widest text-primary">Phân bổ tài sản</CardTitle>
                     </CardHeader>
                     <CardContent className="p-6 space-y-6">
                        {(() => {
                           const holdings = details.productAssetHoldingList && details.productAssetHoldingList.length > 0
                             ? details.productAssetHoldingList.map((asset: any) => ({
                                 name: asset.assetType?.name || 'Khác',
                                 percent: asset.assetPercent
                               }))
                             : details.productAssetAllocationModelList && details.productAssetAllocationModelList.length > 0
                             ? details.productAssetAllocationModelList.map((model: any) => ({
                                 name: model.name || 'Khác',
                                 percent: 0 // API doesn't always provide percent in this list, but we show the categories
                               }))
                             : [];

                           if (holdings.length === 0) {
                             return (
                               <div className="py-10 text-center text-muted-foreground text-xs italic">
                                 Dữ liệu phân bổ đang được cập nhật...
                               </div>
                             );
                           }

                           return holdings.map((asset: any, idx: number) => (
                             <div key={idx} className="space-y-2">
                               <div className="flex justify-between text-xs font-bold">
                                 <span>{asset.name}</span>
                                 {asset.percent > 0 && <span className="text-primary font-black">{asset.percent}%</span>}
                               </div>
                               <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                                 <div className="h-full bg-primary transition-all duration-1000" style={{ width: `${asset.percent || 100}%` }} />
                               </div>
                             </div>
                           ));
                        })()}
                     </CardContent>
                  </Card>

                  <Card className="shadow-xl border-border/50 overflow-hidden">
                     <CardHeader className="bg-muted/50 border-b border-border/50">
                       <CardTitle className="text-sm font-black uppercase tracking-widest text-primary">Top 10 Danh mục</CardTitle>
                     </CardHeader>
                     <CardContent className="p-0">
                       {(() => {
                         const holdings = [
                           ...(details?.productTopHoldingList || []),
                           ...(details?.productTopHoldingBondList || [])
                         ];
                         
                         if (holdings.length === 0) {
                           return (
                             <div className="py-20 text-center text-muted-foreground text-xs italic">
                               Danh mục cổ phiếu/trái phiếu đang được cập nhật...
                             </div>
                           );
                         }

                         return (
                           <Table>
                             <TableHeader className="bg-muted/30">
                                <TableRow>
                                   <TableHead className="text-[10px] uppercase font-bold">Mã CK</TableHead>
                                   <TableHead className="text-[10px] uppercase font-bold">Ngành</TableHead>
                                   <TableHead className="text-right text-[10px] uppercase font-bold">Tỷ trọng</TableHead>
                                </TableRow>
                             </TableHeader>
                             <TableBody>
                               {holdings.map((holding: any, idx: number) => (
                                 <TableRow key={holding.id || idx} className="hover:bg-muted/30 border-b border-border/50 last:border-0 hover:border-l-4 hover:border-l-primary transition-all">
                                   <TableCell className="py-3 text-sm font-black text-primary">{holding.stockCode}</TableCell>
                                   <TableCell className="py-3 text-[11px] text-muted-foreground font-medium">{holding.industry}</TableCell>
                                   <TableCell className="py-3 text-right text-sm font-bold">{holding.netAssetPercent}%</TableCell>
                                 </TableRow>
                               ))}
                             </TableBody>
                           </Table>
                         );
                       })()}
                     </CardContent>
                  </Card>
               </div>
            </TabsContent>
          </Tabs>
        </div>
      </div>
    </div>
  );
}
