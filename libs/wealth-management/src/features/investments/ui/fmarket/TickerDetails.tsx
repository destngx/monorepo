import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp, RefreshCcw } from 'lucide-react';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';

interface TickerDetailsProps {
  details: any;
  history: any[];
  loading: boolean;
  range: any;
  onRangeChange: (r: any) => void;
  onBack: () => void;
}

export function TickerDetails({ 
  details, 
  history, 
  loading, 
  range,
  onRangeChange,
  onBack 
}: TickerDetailsProps) {
  if (loading && !history.length) return <div className="p-10 text-center flex flex-col items-center gap-4"><Skeleton className="h-40 w-full rounded-2xl" /><Skeleton className="h-80 w-full rounded-2xl" /></div>;

  const formattedHistory = (history || []).map(item => {
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
                <CardContent className="h-[400px] min-h-[400px] pt-6 pr-0 relative min-w-0 min-h-0">
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
                                 percent: 0
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
