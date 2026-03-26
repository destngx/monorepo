"use client";

import { Check, X, Sparkles } from "lucide-react";
import { MaskedBalance } from "@wealth-management/ui";
import { CategoryBadge } from "@wealth-management/ui";
import { 
  Tooltip, 
  TooltipContent, 
  TooltipProvider, 
  TooltipTrigger 
} from "@wealth-management/ui";
import { ProposedTransaction } from "../hooks/useNotificationProcessor";

interface NotificationReviewTableProps {
  proposedTransactions: ProposedTransaction[];
  approvedIndices: Set<number>;
  itemStatus: Record<number, 'pending' | 'saving' | 'done' | 'error'>;
  step: string;
  exchangeRate: number;
  toggleApproval: (index: number) => void;
  toggleAll: () => void;
}

export function NotificationReviewTable({
  proposedTransactions,
  approvedIndices,
  itemStatus,
  step,
  exchangeRate,
  toggleApproval,
  toggleAll
}: NotificationReviewTableProps) {
  return (
    <div className="border rounded-2xl bg-card overflow-hidden shadow-sm">
      <div className="max-h-[500px] overflow-auto">
        <table className="w-full text-sm border-collapse">
          <thead className="bg-muted/50 sticky top-0 z-10 backdrop-blur-sm border-b">
            <tr>
              <th className="w-12 p-4 text-center">
                <input 
                  type="checkbox" 
                  className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600 cursor-pointer"
                  checked={approvedIndices.size === proposedTransactions.length && proposedTransactions.length > 0}
                  onChange={toggleAll}
                  disabled={step === 'saving'}
                />
              </th>
              <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Details</th>
              <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Category</th>
              <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Memo</th>
              <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Tag</th>
              <th className="text-right p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Amount</th>
              <th className="text-center p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">Status</th>
            </tr>
          </thead>
          <tbody>
            {proposedTransactions.map((tx, i) => (
              <tr 
                key={i} 
                className={`group border-b last:border-0 transition-all cursor-pointer hover:bg-muted/30 ${
                  !approvedIndices.has(i) && step !== 'saving' ? "bg-muted/10" : ""
                }`}
                onClick={() => toggleApproval(i)}
              >
                <td className="p-4 text-center" onClick={(e) => e.stopPropagation()}>
                  <input 
                    type="checkbox" 
                    className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600 cursor-pointer"
                    checked={approvedIndices.has(i)}
                    onChange={() => toggleApproval(i)}
                    disabled={step === 'saving'}
                  />
                </td>
                <td className="p-4">
                  <div className="flex flex-col">
                    <span className="font-bold text-[10px] text-indigo-600 mb-0.5 uppercase tracking-wide">{tx.accountName}</span>
                    <span className="font-semibold text-sm leading-tight mb-1">{tx.payee}</span>
                    <span className="text-[10px] text-muted-foreground font-medium">
                      {new Date(tx.date).toLocaleDateString(undefined, { day: '2-digit', month: 'short', year: 'numeric' })}
                    </span>
                  </div>
                </td>
                <td className="p-4">
                  <CategoryBadge category={tx.category} type={tx.categoryType} />
                </td>
                <td className="p-4"><span className="text-[10px] text-muted-foreground/50 italic">Empty</span></td>
                <td className="p-4"><span className="text-[10px] text-muted-foreground/50 italic">None</span></td>
                <td className="p-4 text-right">
                  <div className={`font-black text-base tabular-nums whitespace-nowrap ${tx.type === 'deposit' ? 'text-emerald-600' : 'text-foreground'}`}>
                    {tx.accountName.toLowerCase().includes('binance') ? (
                      <TooltipProvider delayDuration={0}>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <span className="cursor-help underline decoration-dashed decoration-muted-foreground/50 underline-offset-4">
                              {tx.type === 'deposit' ? '+' : '-'}<MaskedBalance amount={tx.amount * exchangeRate} />
                            </span>
                          </TooltipTrigger>
                          <TooltipContent side="top" className="max-w-xs text-xs font-mono break-all font-medium py-2">
                            <div className="text-[10px] text-muted-foreground mb-1 font-sans font-semibold uppercase tracking-wider">Formula Preview</div>
                            ={tx.amount} * exchangeRate
                            <div className="mt-2 text-[10px] text-muted-foreground font-sans">
                              Rate: <MaskedBalance amount={exchangeRate} />
                            </div>
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    ) : (
                      <>{tx.type === 'deposit' ? '+' : '-'}<MaskedBalance amount={tx.amount} /></>
                    )}
                  </div>
                </td>
                <td className="p-4">
                  <div className="flex items-center justify-center min-w-[80px]">
                    {itemStatus[i] === 'saving' && (
                      <div className="flex items-center gap-2 px-2 py-1 rounded-full bg-primary/10 text-primary animate-pulse">
                        <Sparkles className="h-3 w-3 animate-pulse" />
                        <span className="text-[10px] font-bold uppercase tracking-wider">Thinking...</span>
                      </div>
                    )}
                    {itemStatus[i] === 'done' && (
                      <div className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-emerald-100 text-emerald-700">
                        <Check className="h-3 w-3 stroke-[3]" />
                        <span className="text-[10px] font-bold uppercase">Done</span>
                      </div>
                    )}
                    {itemStatus[i] === 'error' && (
                      <div className="flex items-center gap-1.5 px-2 py-1 rounded-full bg-red-100 text-red-700">
                        <X className="h-3 w-3 stroke-[3]" />
                        <span className="text-[10px] font-bold uppercase">Error</span>
                      </div>
                    )}
                    {itemStatus[i] === 'pending' && (
                      <span className="text-[9px] font-bold text-muted-foreground/60 uppercase tracking-widest group-hover:text-primary transition-colors">Pending</span>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
