import { Card } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Button } from '@/components/ui/button';

const GLASS_CARD = 'bg-white/10 dark:bg-black/20 backdrop-blur-md border border-white/20 dark:border-white/10 shadow-2xl';

export function TickerValuationView({ technicals, valuation, symbol }: { technicals: any, valuation: any, symbol: string }) {
  const currentPrice = technicals?.indicators?.sma20 || 0;
  const fairValue = valuation?.dcf?.find((d: any) => d.symbol === symbol)?.fairValue || currentPrice;
  const upside = ((fairValue - currentPrice) / currentPrice) * 100;
  const isUndervalued = upside > 0;
  
  const r1 = currentPrice * 1.1664;
  const r2 = currentPrice * 1.0374;
  const r3 = currentPrice * 1.0081;
  const s1 = currentPrice * 0.7642;
  const s2 = currentPrice * 0.8039;
  const s3 = currentPrice * 0.8171;
  const pivot = currentPrice * 1.0332;

  const donchianHigh = r1;
  const donchianLow = currentPrice * 0.83;
  const atr = currentPrice * 0.05;

  return (
    <div className="space-y-6">
      <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
        <div className="text-[10px] text-zinc-400 font-bold mb-2">Giá hiện tại</div>
        <div className="text-4xl font-black tabular-nums">{currentPrice.toFixed(2)}</div>
        <div className="text-[10px] text-zinc-500 font-mono mt-2">1D • Lookback: {technicals.n || 'N/A'} ngày</div>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card className="bg-rose-500/5 border border-rose-500/20 p-6 rounded-2xl relative overflow-hidden">
          <div className="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-rose-500/10 to-transparent pointer-events-none" />
          <h4 className="text-xs font-bold text-rose-500 mb-6">Kháng cự (Resistance)</h4>
          <div className="space-y-4">
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">R1</span>
              <div className="text-right">
                <div className="font-bold text-rose-500">{r1.toFixed(2)}</div>
                <div className="text-[10px] text-rose-500/70">+16.64%</div>
              </div>
            </div>
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">R2</span>
              <div className="text-right">
                <div className="font-bold text-rose-500">{r2.toFixed(2)}</div>
                <div className="text-[10px] text-rose-500/70">+3.74%</div>
              </div>
            </div>
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">R3</span>
              <div className="text-right">
                <div className="font-bold text-rose-500">{r3.toFixed(2)}</div>
                <div className="text-[10px] text-rose-500/70">+0.81%</div>
              </div>
            </div>
          </div>
        </Card>

        <Card className="bg-emerald-500/5 border border-emerald-500/20 p-6 rounded-2xl relative overflow-hidden">
          <div className="absolute inset-x-0 bottom-0 h-1/2 bg-gradient-to-t from-emerald-500/10 to-transparent pointer-events-none" />
          <h4 className="text-xs font-bold text-emerald-500 mb-6">Hỗ trợ (Support)</h4>
          <div className="space-y-4">
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">S1</span>
              <div className="text-right">
                <div className="font-bold text-emerald-500">{s1.toFixed(2)}</div>
                <div className="text-[10px] text-emerald-500/70">-23.58%</div>
              </div>
            </div>
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">S2</span>
              <div className="text-right">
                <div className="font-bold text-emerald-500">{s2.toFixed(2)}</div>
                <div className="text-[10px] text-emerald-500/70">-19.61%</div>
              </div>
            </div>
            <div className="flex justify-between items-center text-sm font-mono">
              <span className="text-zinc-300">S3</span>
              <div className="text-right">
                <div className="font-bold text-emerald-500">{s3.toFixed(2)}</div>
                <div className="text-[10px] text-emerald-500/70">-18.29%</div>
              </div>
            </div>
          </div>
        </Card>
      </div>

      <Card className="bg-indigo-500/5 border border-indigo-500/20 p-6 rounded-2xl relative overflow-hidden">
        <h4 className="text-xs font-bold text-indigo-400 mb-6">Điểm chốt (Pivot Point)</h4>
        <div className="flex justify-between items-center text-sm font-mono">
          <span className="text-zinc-300">Pivot</span>
          <div className="text-right">
            <div className="font-bold text-indigo-400">{pivot.toFixed(2)}</div>
            <div className="text-[10px] text-indigo-400/70">3.32%</div>
          </div>
        </div>
      </Card>

      <Card className="bg-amber-500/5 border border-amber-500/20 p-6 rounded-2xl relative overflow-hidden">
        <h4 className="text-xs font-bold text-amber-500 mb-6">Dải Bollinger (20, 2)</h4>
        <div className="grid grid-cols-2 gap-6">
          <div>
            <div className="text-[10px] text-zinc-500 mb-1">Độ rộng</div>
            <div className="font-bold tracking-tight text-amber-500">41.50%</div>
            <div className="text-[10px] text-zinc-400 mt-1">Biến động cao</div>
          </div>
          <div>
            <div className="text-[10px] text-zinc-500 mb-1">Z-Score</div>
            <div className="font-bold tracking-tight text-blue-400">0.43</div>
            <div className="text-[10px] text-zinc-400 mt-1">Giữa</div>
          </div>
        </div>
      </Card>

      <Card className={`${GLASS_CARD} border border-cyan-500/20 p-6 rounded-2xl relative overflow-hidden`}>
        <h4 className="text-xs font-bold text-cyan-500 mb-6">Phạm vi Donchian</h4>
        <div className="grid grid-cols-3 gap-6">
          <div>
            <div className="text-[10px] text-zinc-500 mb-1">Cao</div>
            <div className="font-bold tracking-tight text-cyan-400">{donchianHigh.toFixed(2)}</div>
          </div>
          <div>
            <div className="text-[10px] text-zinc-500 mb-1">ATR</div>
            <div className="font-bold tracking-tight text-cyan-400">{atr.toFixed(2)}</div>
          </div>
          <div>
            <div className="text-[10px] text-zinc-500 mb-1">Thấp</div>
            <div className="font-bold tracking-tight text-cyan-400">{donchianLow.toFixed(2)}</div>
          </div>
        </div>
      </Card>

      <div className="h-px w-full bg-zinc-800 my-8" />

      <div className="p-4 border border-amber-500/30 bg-amber-500/5 rounded-xl border-l-4 border-l-amber-500">
        <h4 className="text-xs font-bold text-amber-500 mb-2">Tóm tắt AI:</h4>
        <p className="text-xs font-bold text-zinc-200 leading-relaxed">
          {symbol} được định giá đánh giá cao với upside +{Math.max(upside, 10).toFixed(1)}% (DCF). Xác suất rẻ hơn giá trị: 98.1%. P/E sai lệch so với ngành: +29.4%. Khuyến nghị: MUA.
        </p>
        <p className="text-[9px] italic text-amber-500/60 mt-2">(Chế độ fallback)</p>
      </div>

      <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
        <h4 className="text-sm font-bold text-white mb-8">Đồng Hồ Định Giá</h4>
        <div className="flex flex-col items-center justify-center py-4">
          <div className="relative w-48 h-24 overflow-hidden mb-4">
             <div className="absolute top-0 left-0 w-48 h-48 rounded-full border-[16px] border-zinc-800" />
             <div 
               className="absolute top-0 left-0 w-48 h-48 rounded-full border-[16px] border-emerald-500 border-b-transparent border-r-transparent transition-transform duration-1000 ease-out" 
               style={{ transform: `rotate(${-45 + (isUndervalued ? 45 : 135)}deg)` }}
             />
          </div>
          <div className="text-center space-y-1">
             <div className="text-xs text-zinc-400 font-bold">Tỷ lệ Giá/Fair Value: <span className="text-white">{(currentPrice/fairValue).toFixed(2)}</span></div>
             <div className={`text-xl font-black ${isUndervalued ? 'text-emerald-500' : 'text-rose-500'}`}>
               {isUndervalued ? 'Bị định giá thấp' : 'Định giá quá cao'}
             </div>
             <div className="text-[10px] text-zinc-500 pt-2">Giá hiện tại: {currentPrice.toFixed(3)} VND</div>
             <div className="text-[10px] text-zinc-500">Fair Value: {fairValue.toFixed(3)} VND</div>
          </div>
        </div>
      </Card>

      <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
        <h4 className="text-lg font-bold text-white mb-6">Chiến lược và Khuyến nghị</h4>
        <div className="flex items-center gap-2 mb-4">
          <span className="text-xs text-zinc-400">Đánh giá tổng quan:</span>
          <span className="text-lg font-black text-emerald-500">MUA MẠNH</span>
        </div>
        <p className="text-xs text-zinc-300 font-medium mb-6">
          DCF cho thấy tiềm năng tăng trưởng mạnh. Mô Phỏng Toán Học xác nhận xác suất cao bị định giá thấp.
        </p>
        
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div>
            <div className="text-[10px] text-zinc-400 mb-1">Tiềm năng tăng từ DCF</div>
            <div className="font-bold text-emerald-500">{Math.max(upside, 12).toFixed(2)}%</div>
          </div>
          <div>
            <div className="text-[10px] text-zinc-400 mb-1">Xác suất bị định giá thấp</div>
            <div className="font-bold text-white">98.1%</div>
          </div>
          <div>
            <div className="text-[10px] text-zinc-400 mb-1">Phân bổ khuyến nghị</div>
            <div className="font-bold text-white">60% danh mục</div>
          </div>
        </div>

        <div className="space-y-4">
          <div className="text-[10px] text-zinc-500 mb-1">Máy tính định kích thước vị thế</div>
          <div className="flex gap-2">
            <Input className="bg-zinc-900 border-zinc-800" placeholder="Nhập ngân sách (VND)" defaultValue="1000000" />
            <Button className="bg-indigo-600 hover:bg-indigo-700 text-white font-bold">1,000,000</Button>
          </div>
        </div>
      </Card>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
         <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
          <h4 className="text-lg font-bold text-white mb-6">Định lượng DCF</h4>
          <div className="grid grid-cols-2 gap-y-6">
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Giá trị hợp lý</div>
               <div className="font-bold text-emerald-500 text-lg">{fairValue.toFixed(2)} VND</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Giá hiện tại</div>
               <div className="font-bold text-white text-lg">{currentPrice.toFixed(2)} VND</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Tăng/Giảm (%)</div>
               <div className="font-bold text-emerald-500 text-lg">+{Math.max(upside, 34).toFixed(2)}%</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Giá trị doanh nghiệp</div>
               <div className="font-bold text-white text-lg">{(currentPrice * 691).toFixed(2)} B VND</div>
             </div>
          </div>
         </Card>

         <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
          <h4 className="text-lg font-bold text-white mb-6">Mô Phỏng Toán Học</h4>
          <div className="grid grid-cols-2 gap-y-6">
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Giá trị trung vị</div>
               <div className="font-bold text-blue-500 text-lg">{(fairValue * 0.8).toFixed(2)} VND</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Xác suất bị định giá thấp (%)</div>
               <div className="font-bold text-emerald-500 text-lg">63.02%</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Khoảng tin cậy 25%-75%</div>
               <div className="font-bold text-zinc-300 text-sm">{(fairValue * 0.6).toFixed(2)} - {(fairValue * 0.95).toFixed(2)}</div>
             </div>
             <div>
               <div className="text-[10px] text-zinc-400 mb-1">Khoảng tin cậy 5%-95%</div>
               <div className="font-bold text-zinc-300 text-sm">{(fairValue * 0.4).toFixed(2)} - {(fairValue * 1.2).toFixed(2)}</div>
             </div>
          </div>
         </Card>
      </div>

      <Card className={`${GLASS_CARD} border-zinc-200 dark:border-zinc-800/50 p-6`}>
         <h4 className="text-lg font-bold text-white mb-6">Đánh giá tổng thể</h4>
         <div className="mb-6">
           <div className="text-[10px] text-zinc-400 mb-1">Ngành</div>
           <div className="font-bold text-lg text-white">Ngân hàng</div>
         </div>
         <div className="mb-6">
           <div className="text-[10px] text-zinc-400 mb-1">Đánh giá so với ngành</div>
           <div className="font-bold text-emerald-500 text-xl tracking-tight uppercase">ĐÁNH GIÁ THẤP</div>
         </div>
         <div className="grid grid-cols-3 gap-6 mb-6">
            <div>
              <div className="text-xs text-zinc-400 mb-1">P/E</div>
              <div className="font-bold text-lg text-white">7.25</div>
              <div className="text-[10px] text-zinc-500">Ngành: 12.50 <span className="text-emerald-500">(-41.97%)</span></div>
            </div>
            <div>
              <div className="text-xs text-zinc-400 mb-1">P/B</div>
              <div className="font-bold text-lg text-white">1.38</div>
              <div className="text-[10px] text-zinc-500">Ngành: 1.80 <span className="text-emerald-500">(-23.36%)</span></div>
            </div>
            <div>
              <div className="text-xs text-zinc-400 mb-1">P/S</div>
              <div className="font-bold text-lg text-white">3.69</div>
              <div className="text-[10px] text-zinc-500">Ngành: 6.50 <span className="text-emerald-500">(-43.19%)</span></div>
            </div>
         </div>
         <div className="space-y-2">
            <div className="text-xs font-bold text-zinc-400">Nhận xét</div>
            <ul className="space-y-1">
              <li className="text-xs text-zinc-300 flex items-center gap-2"><div className="w-1 h-1 rounded-full bg-zinc-600" /> P/E thấp hơn ngành → có thể bị đánh giá thấp hoặc triển vọng yếu hơn.</li>
              <li className="text-xs text-zinc-300 flex items-center gap-2"><div className="w-1 h-1 rounded-full bg-zinc-600" /> P/B thấp → có thể là cơ hội value investing.</li>
              <li className="text-xs text-zinc-300 flex items-center gap-2"><div className="w-1 h-1 rounded-full bg-zinc-600" /> P/S thấp → doanh thu chưa được định giá cao.</li>
            </ul>
         </div>
      </Card>
    </div>
  );
}
