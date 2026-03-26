import { Card, CardHeader, CardTitle, CardContent, MaskedBalance, AIDataInsight, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@wealth-management/ui';
import { RefreshCcw, Bitcoin, TrendingUp } from 'lucide-react';

interface AssetData {
  headers: string[];
  holdings: any[];
}

interface AssetLedgersProps {
  assets: {
    crypto: AssetData;
    funds: AssetData;
  } | null;
  isFetchingPrices: boolean;
  prices: Record<string, number>;
  exchangeRate: number;
}

export function AssetLedgers({ assets, isFetchingPrices, prices, exchangeRate }: AssetLedgersProps) {
  const renderAssetTable = (title: string, data?: AssetData, icon?: React.ReactNode) => {
    if (!data || data.holdings.length === 0) {
      return (
        <Card className="shadow-sm border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              {icon} {title}
            </CardTitle>
          </CardHeader>
          <CardContent className="flex items-center justify-center p-8 text-muted-foreground bg-muted/20 border-t rounded-b-xl border-dashed">
            No specific data tracked in sheets.
          </CardContent>
        </Card>
      );
    }

    const hasVndColumn = data.headers.some((h) => {
      const lowH = h.toLowerCase();
      return lowH.includes('vnd') && (lowH.includes('value') || lowH.includes('total') || lowH.includes('balance'));
    });

    const displayHeaders = hasVndColumn ? data.headers : [...data.headers, 'Est. VND Value'];

    const columnTotals: (number | null)[] = displayHeaders.map((header) => {
      const isVirtualVnd = header === 'Est. VND Value';
      if (!isVirtualVnd) return null;

      let sum = 0;
      let hasAny = false;

      for (const row of data.holdings) {
        const val = row[header];

        const amountKey = Object.keys(row).find((k) => {
          const lk = k.toLowerCase();
          return (
            lk.includes('amount') ||
            lk.includes('qty') ||
            lk.includes('quantity') ||
            lk.includes('unit') ||
            lk.includes('holding') ||
            lk.includes('balance') ||
            lk.includes('total') ||
            lk.includes('coin') ||
            lk.includes('coins') ||
            lk.includes('units') ||
            k.match(/^(amount|qty|quantity|holding|balance|total|coin|coins|units)$/i)
          );
        });
        const priceKey = Object.keys(row).find((k) => {
          const lk = k.toLowerCase();
          return lk.includes('price') || lk.includes('rate') || lk.includes('cost');
        });
        const symbol =
          row['Token'] || row['Currency'] || row['Asset'] || row['Certificate'] || row['Index'] || row['Symbol'];
        const sheetPrice = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
        const aiPrice = prices[String(symbol)] || 0;
        const finalPrice = !isNaN(sheetPrice) && sheetPrice !== 0 ? sheetPrice : aiPrice;

        let displayVal = val;
        if (!val || val === '0' || val === 0 || isVirtualVnd) {
          if (amountKey && row[amountKey] && finalPrice > 0) {
            const amount = parseFloat(String(row[amountKey]).replace(/,/g, ''));
            if (!isNaN(amount)) {
              let calculated = amount * finalPrice;
              if (title.toLowerCase().includes('crypto') && finalPrice < 100000) {
                calculated *= exchangeRate;
              }
              displayVal = calculated;
            }
          }
        }

        const num =
          displayVal !== undefined && displayVal !== null && displayVal !== ''
            ? parseFloat(String(displayVal).replace(/,/g, ''))
            : NaN;

        if (!isNaN(num) && num !== 0) {
          sum += num;
          hasAny = true;
        }
      }

      return hasAny ? parseFloat(sum.toFixed(2)) : null;
    });

    const isCryptoTable = title.toLowerCase().includes('crypto');
    const isIfcTable =
      title.toLowerCase().includes('fund') ||
      title.toLowerCase().includes('certificate') ||
      title.toLowerCase().includes('ifc');

    return (
      <Card className="shadow-sm border-border/50 overflow-hidden text-zinc-900 dark:text-zinc-100">
        <CardHeader className="bg-muted/30 border-b flex flex-row items-center justify-between py-3">
          <div className="flex items-center gap-2">
            <CardTitle className="text-lg flex flex-row items-center gap-2 m-0 p-0">
              {icon} {title}
            </CardTitle>
            <AIDataInsight
              type={`${title} Table`}
              description={`Detailed list of ${title.toLowerCase()} including balance, price, and estimated value.`}
              data={data.holdings}
            />
          </div>
          {isFetchingPrices && (
            <div className="flex items-center gap-2 text-[10px] text-primary animate-pulse font-medium bg-primary/10 px-2 py-0.5 rounded-full">
              <RefreshCcw className="w-3 h-3 animate-spin" />
              Refreshing AI Prices...
            </div>
          )}
        </CardHeader>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            {displayHeaders && displayHeaders.length > 0 && displayHeaders[0] !== '' && (
              <thead className="bg-muted/50 border-b text-muted-foreground uppercase tracking-wider text-[10px] font-semibold">
                <tr>
                  {displayHeaders.map((h, i) => (
                    <th key={i} className="text-left py-3 px-4 whitespace-nowrap">
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
            )}
            <tbody className="divide-y">
              {data.holdings.map((row, i) => (
                <tr key={i} className="hover:bg-muted/25 transition-colors">
                  {displayHeaders.map((header, j) => (
                    <td key={j} className="py-3 px-4 whitespace-nowrap">
                      {(() => {
                        const h = header.toLowerCase();
                        const val = row[header];

                        const isVirtualVnd = header === 'Est. VND Value';
                        const sensitiveHeaders = [
                          'amount', 'balance', 'vnd', 'unit', 'holding', 'qty', 'quantity', 'price', 'cost', 'value', 'usd', 'total'
                        ];

                        let displayVal = val;
                        const isVndTotalCell = h.includes('vnd') || h === 'total' || h === 'value' || isVirtualVnd;

                        const amountKey = Object.keys(row).find((k) => {
                          const lk = k.toLowerCase();
                          return (
                            lk.includes('amount') || lk.includes('qty') || lk.includes('quantity') || lk.includes('unit') || lk.includes('holding') || lk.includes('balance') || lk.includes('total') || lk.includes('coin') || lk.includes('coins') || lk.includes('units') || k.match(/^(amount|qty|quantity|holding|balance|spot|coin|coins|units)$/i)
                          );
                        });
                        const priceKey = Object.keys(row).find(k => k.toLowerCase().includes('price') || k.toLowerCase().includes('rate') || k.toLowerCase().includes('cost'));

                        const symbol = row['Token'] || row['Currency'] || row['Asset'] || row['Certificate'] || row['Index'] || row['Symbol'];
                        const sheetPrice = priceKey ? parseFloat(String(row[priceKey]).replace(/,/g, '')) : 0;
                        const aiPrice = prices[String(symbol)] || 0;
                        const finalPrice = !isNaN(sheetPrice) && sheetPrice !== 0 ? sheetPrice : aiPrice;
                        const isPriceAI = (!sheetPrice || sheetPrice === 0) && aiPrice > 0;

                        if (isVndTotalCell && (!val || val === '0' || val === 0 || isVirtualVnd)) {
                          if (amountKey && row[amountKey] && finalPrice > 0) {
                            const amount = parseFloat(String(row[amountKey]).replace(/,/g, ''));
                            if (!isNaN(amount)) {
                              let calculated = amount * finalPrice;
                              if (title.toLowerCase().includes('crypto') && finalPrice < 100000) calculated *= exchangeRate;
                              displayVal = calculated;
                            }
                          }
                        }

                        const isNum = displayVal && (typeof displayVal === 'number' || (typeof displayVal === 'string' && /^-?\d+(,\d+)*(\.\d+)?$/.test(displayVal.trim())));

                        if (isNum || sensitiveHeaders.some((sh) => h.includes(sh))) {
                          const unit = row['Token'] || row['Currency'] || row['Asset'] || row['Certificate'] || row['Index'] || row['Symbol'];
                          const isCalculated = isVndTotalCell && (!val || val === '0' || val === 0 || isVirtualVnd);
                          const isVndCell = h.includes('vnd') || isVirtualVnd;
                          const content = (
                            <MaskedBalance
                              value={displayVal}
                              unit={!isVndCell && typeof unit === 'string' && unit !== displayVal ? unit : undefined}
                              currency={isVndCell ? 'VND' : h.includes('usd') ? 'USD' : 'none'}
                            />
                          );

                          if (isCalculated && (isVirtualVnd || isPriceAI)) {
                             const amount = amountKey ? parseFloat(String(row[amountKey]).replace(/,/g, '')) : NaN;
                             const priceLabel = title.toLowerCase().includes('fund') ? 'NAV Rate' : isPriceAI ? 'AI Fetched Price' : priceKey;
                             const rateUsed = title.toLowerCase().includes('crypto') && finalPrice < 100000 ? exchangeRate : 1;

                             return (
                               <TooltipProvider>
                                 <Tooltip>
                                   <TooltipTrigger asChild><span className="cursor-help border-b border-dotted border-primary/30 pb-0.5">{content}</span></TooltipTrigger>
                                   <TooltipContent className="text-[10px] font-mono p-2">
                                     <div className="flex flex-col gap-1">
                                       <div className="text-muted-foreground uppercase text-[9px] font-bold">Formula</div>
                                       {!isNaN(amount) && <div>{amount.toLocaleString()} ({amountKey})</div>}
                                       <div className={isPriceAI ? 'text-indigo-500 font-bold' : ''}>× {finalPrice.toLocaleString()} ({priceLabel})</div>
                                       {rateUsed !== 1 && <div>× {rateUsed.toLocaleString()} (USDT Rate)</div>}
                                       <div className="border-t pt-1 mt-1 font-bold text-indigo-600 dark:text-indigo-400">= {Number(displayVal).toLocaleString()} VND</div>
                                     </div>
                                   </TooltipContent>
                                 </Tooltip>
                               </TooltipProvider>
                             );
                          }
                          return content;
                        }
                        return displayVal || '';
                      })()}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
            <tfoot className={`border-t-2 font-semibold text-sm ${isCryptoTable ? 'bg-orange-500/10 border-orange-500/30' : isIfcTable ? 'bg-emerald-500/10 border-emerald-500/30' : 'bg-muted/40'}`}>
              <tr>
                {displayHeaders.map((header, j) => {
                  if (j === 0) return <td key={j} className="py-3 px-4"><span className={`font-bold text-xs uppercase tracking-wide ${isCryptoTable ? 'text-orange-600' : isIfcTable ? 'text-emerald-600' : 'text-muted-foreground'}`}>{isCryptoTable ? 'Total Crypto' : isIfcTable ? 'Total IFC' : 'Total'}</span></td>;
                  const total = columnTotals[j];
                  if (header === 'Est. VND Value' && total !== null) return <td key={j} className="py-3 px-4 whitespace-nowrap font-bold"><MaskedBalance value={total} currency="VND" /></td>;
                  return <td key={j} className="py-3 px-4" />;
                })}
              </tr>
            </tfoot>
          </table>
        </div>
      </Card>
    );
  };

  return (
    <div className="space-y-6">
      {renderAssetTable('Crypto Holdings', assets?.crypto, <Bitcoin className="h-5 w-5 text-orange-500" />)}
      {renderAssetTable('Investment Fund Certificates', assets?.funds, <TrendingUp className="h-5 w-5 text-emerald-500" />)}
    </div>
  );
}
