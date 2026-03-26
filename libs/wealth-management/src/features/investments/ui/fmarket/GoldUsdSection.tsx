import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card';
import { Skeleton } from '@/components/ui/skeleton';
import { TrendingUp, Landmark, Award } from 'lucide-react';
import { ResponsiveContainer, AreaChart, CartesianGrid, XAxis, YAxis, Tooltip, Legend, Area } from 'recharts';
import { GoldProductTable } from './GoldProductTable';

interface GoldUsdSectionProps {
  loading: boolean;
  goldLoading: boolean;
  usdLoading: boolean;
  goldRange: string;
  setGoldRange: (r: any) => void;
  usdRange: string;
  setUsdRange: (r: any) => void;
  formattedGoldData: any[];
  formattedUsdData: any[];
  goldProducts: any[];
}

export function GoldUsdSection({ 
  loading, 
  goldLoading, 
  usdLoading, 
  goldRange, 
  setGoldRange, 
  usdRange, 
  setUsdRange, 
  formattedGoldData, 
  formattedUsdData,
  goldProducts 
}: GoldUsdSectionProps) {
  return (
    <div className="space-y-6 animate-in fade-in duration-300">
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        <div className="lg:col-span-1 space-y-6">
          <Card className="shadow-sm border-border/50 overflow-hidden bg-gradient-to-br from-amber-500/5 to-transparent">
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-black uppercase tracking-widest text-amber-600 flex items-center gap-2">
                <TrendingUp className="h-4 w-4" /> Bảng giá Vàng trực tuyến
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4 text-zinc-900 dark:text-zinc-100">
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
                  <span className="text-lg font-black text-blue-600 font-mono">145,093,896</span>
                  <span className="text-[10px] font-bold text-muted-foreground">VND</span>
                </div>
              </div>

              <div className="pt-2 border-t border-border/50">
                <div className="text-[10px] font-bold text-muted-foreground uppercase opacity-70">Tỷ giá USD/VND</div>
                <div className="flex items-baseline gap-2">
                  <span className="text-lg font-black text-blue-700 font-mono">26,359</span>
                  <span className="text-[10px] font-bold text-muted-foreground">VND</span>
                </div>
              </div>

              <div className="text-[10px] text-right font-medium text-muted-foreground mt-4 italic">
                Cập nhật Ngày 25/03/2026
              </div>
            </CardContent>
          </Card>
        </div>

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
                      goldRange === r ? 'bg-white text-primary shadow-sm' : 'text-muted-foreground hover:text-foreground'
                    }`}
                  >
                    {r}
                  </button>
                ))}
              </div>
            </div>
          </CardHeader>
          <CardContent className="h-[220px] min-h-[220px] pt-4 pr-0 relative min-w-0 min-h-0">
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
                  <XAxis dataKey="date" fontSize={9} tickLine={false} axisLine={false} minTickGap={40} />
                  <YAxis fontSize={9} tickLine={false} axisLine={false} domain={['auto', 'auto']} tickFormatter={(val) => `${val}M`} />
                  <Tooltip contentStyle={{ fontSize: '11px', borderRadius: '12px', border: 'none', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }} />
                  <Legend verticalAlign="top" height={30} iconType="circle" wrapperStyle={{ fontSize: '10px' }} />
                  <Area type="monotone" dataKey="vnAsk" name="SJC Bán ra (VN)" stroke="#ef4444" strokeWidth={2} fillOpacity={1} fill="url(#colorVnAsk)" />
                  <Area type="monotone" dataKey="vnBid" name="SJC Mua vào (VN)" stroke="#10b981" strokeWidth={2} fillOpacity={1} fill="url(#colorVnBid)" />
                  <Area type="monotone" dataKey="world" name="Thế giới (Quy đổi)" stroke="#3b82f6" strokeWidth={1.5} strokeDasharray="4 4" fillOpacity={1} fill="url(#colorWorld)" />
                </AreaChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

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
                    usdRange === r ? 'bg-white text-primary shadow-sm' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {r}
                </button>
              ))}
            </div>
          </div>
        </CardHeader>
        <CardContent className="h-[220px] min-h-[220px] pt-4 pr-0 relative min-w-0 min-h-0">
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
                <XAxis dataKey="date" fontSize={9} tickLine={false} axisLine={false} minTickGap={40} />
                <YAxis fontSize={9} tickLine={false} axisLine={false} domain={['auto', 'auto']} tickFormatter={(val) => Math.round(val).toLocaleString()} />
                <Tooltip contentStyle={{ fontSize: '11px', borderRadius: '12px', border: 'none', boxShadow: '0 8px 32px rgba(0,0,0,0.12)' }} />
                <Area type="monotone" dataKey="price" name="USD/VND" stroke="#3b82f6" strokeWidth={2} fillOpacity={1} fill="url(#colorUsd)" />
              </AreaChart>
            </ResponsiveContainer>
          )}
        </CardContent>
      </Card>

      <Card className="shadow-sm border-border/50 overflow-hidden">
        <CardHeader className="pb-4 bg-amber-500/5 border-b border-border/50">
          <CardTitle className="text-lg font-bold flex items-center gap-2 text-amber-700">
            < Award className="h-5 w-5 text-amber-500" />
            Bảng giá Vàng trực tuyến
          </CardTitle>
          <CardDescription className="text-xs">
            Thời gian giao dịch vàng từ 9h00 - 18h30 | Cập nhật lúc {new Date().toLocaleTimeString('vi-VN', {hour: '2-digit', minute:'2-digit'})}, {new Date().toLocaleDateString('vi-VN')}
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0 text-zinc-900 dark:text-zinc-100">
          <GoldProductTable products={goldProducts} loading={loading} />
        </CardContent>
      </Card>
    </div>
  );
}
