import { Card, CardHeader, CardTitle, CardDescription, CardContent, Skeleton } from '@wealth-management/ui';
import { Landmark } from 'lucide-react';
import { ResponsiveContainer, BarChart, CartesianGrid, XAxis, YAxis, Tooltip, Bar } from 'recharts';

interface BankRatesSectionProps {
  loading: boolean;
  bankData: any;
  bankRates: any[];
}

export function BankRatesSection({ loading, bankData, bankRates }: BankRatesSectionProps) {
  return (
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
                          <div className="bg-white p-3 border border-border/50 shadow-2xl rounded-xl text-zinc-900">
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
  );
}
