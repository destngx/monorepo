'use client';

import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@wealth-management/ui/dialog';
import { Button } from "@wealth-management/ui/button";
import { Check, AlertCircle, X, Sparkles } from 'lucide-react';
import { MaskedBalance } from '@wealth-management/ui/masked-balance';
import { CategoryBadge } from '@wealth-management/ui/category-badge';
import { EmailNotification } from '@wealth-management/types';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@wealth-management/ui/tooltip';
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface ProposedTransaction {
  notificationId: string;
  date: string;
  payee: string;
  amount: number;
  type: 'payment' | 'deposit';
  accountName: string;
  category: string;
  categoryType?: 'income' | 'expense' | 'non-budget';
  memo: string;
}

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete: () => void;
}

export function NotificationProcessor({ open, onOpenChange, onComplete }: Props) {
  const [step, setStep] = useState<'fetching' | 'parsing' | 'review' | 'saving'>('fetching');
  const [notifications, setNotifications] = useState<EmailNotification[]>([]);
  const [proposedTransactions, setProposedTransactions] = useState<ProposedTransaction[]>([]);
  const [approvedIndices, setApprovedIndices] = useState<Set<number>>(new Set());
  const [error, setError] = useState<string | null>(null);
  const [exchangeRate, setExchangeRate] = useState<number>(25400);

  const [itemStatus, setItemStatus] = useState<Record<number, 'pending' | 'saving' | 'done' | 'error'>>({});

  const reset = () => {
    setStep('fetching');
    setNotifications([]);
    setProposedTransactions([]);
    setApprovedIndices(new Set());
    setItemStatus({});
    setError(null);
  };

  const startProcessing = async () => {
    try {
      setStep('fetching');

      // Fetch current exchange rate for Binance conversions
      void fetch('/api/exchange-rate')
        .then((res) => res.json())
        .then((data) => setExchangeRate(data.rate || 25400))
        .catch(() => {});

      const res = await fetch('/api/notifications');
      const data = await res.json();

      if (data.length === 0) {
        setStep('review');
        setNotifications([]);
        return;
      }

      setNotifications(data);
      setStep('parsing');

      const aiRes = await fetch('/api/ai/parse-notifications', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notifications: data }),
      });

      const parsedTransactions = await aiRes.json();

      if (aiRes.ok && Array.isArray(parsedTransactions)) {
        setProposedTransactions(parsedTransactions);
        setApprovedIndices(new Set(parsedTransactions.map((_: any, i: number) => i)));

        const initialStatus: Record<number, 'pending'> = {};
        parsedTransactions.forEach((_: any, i: number) => {
          initialStatus[i] = 'pending';
        });
        setItemStatus(initialStatus);
        setStep('review');
      } else {
        const errorMessage = parsedTransactions.error || 'Failed to parse notifications. Please try again.';
        setError(errorMessage);
        setStep('review');
      }
    } catch (error: any) {
      const message = getErrorMessage(error);
      const errorMsg = isAppError(error) ? error.userMessage : message;
      setError(errorMsg);
      setStep('review');
    }
  };

  useEffect(() => {
    if (open) {
      void startProcessing();
    } else {
      reset();
    }
  }, [open]);

  const toggleApproval = (index: number) => {
    if (step === 'saving') return;
    const next = new Set(approvedIndices);
    if (next.has(index)) next.delete(index);
    else next.add(index);
    setApprovedIndices(next);
  };

  const toggleAll = () => {
    if (step === 'saving') return;
    if (approvedIndices.size === proposedTransactions.length) {
      setApprovedIndices(new Set());
    } else {
      setApprovedIndices(new Set(proposedTransactions.map((_, i) => i)));
    }
  };

  const handleSave = async () => {
    setStep('saving');
    try {
      // 1. Mark ALL fetched notifications as done in sheets
      const allRowNumbers = notifications
        .map((n) => n.rowNumber)
        .filter((row): row is number => typeof row === 'number');

      if (allRowNumbers.length > 0) {
        await fetch('/api/notifications', {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ rowNumbers: allRowNumbers }),
        });
      }

      // 2. Add only APPROVED transactions
      const indicesToSave = Array.from(approvedIndices).sort((a, b) => a - b);

      for (const index of indicesToSave) {
        const t = proposedTransactions[index];
        setItemStatus((prev) => ({ ...prev, [index]: 'saving' }));

        try {
          await fetch('/api/transactions', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              accountName: t.accountName,
              date: t.date,
              payee: t.payee,
              category: t.category,
              memo: '',
              payment: t.type === 'payment' ? t.amount : null,
              deposit: t.type === 'deposit' ? t.amount : null,
              tags: [],
              cleared: false,
            }),
          });

          setItemStatus((prev) => ({ ...prev, [index]: 'done' }));
        } catch (err) {
          setItemStatus((prev) => ({ ...prev, [index]: 'error' }));
          // Continue with others even if one fails
        }
      }

      onComplete();
      setTimeout(() => onOpenChange(false), 1000);
    } catch (err: any) {
      const message = getErrorMessage(err);
      const errorMsg = isAppError(err) ? err.userMessage : message;
      setError(errorMsg);
      setStep('review');
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[65vw] max-h-[90vh] flex flex-col p-0">
        <DialogHeader className="p-6 pb-2">
          <DialogTitle className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-indigo-500" />
            Import from Email
          </DialogTitle>
          <DialogDescription>
            AI is analyzing your bank notification emails to suggest new transactions.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 px-6 pb-6 overflow-hidden">
          {step === 'fetching' || step === 'parsing' ? (
            <div className="h-[400px] flex flex-col items-center justify-center gap-4">
              <div className="relative">
                <Sparkles className="h-10 w-10 text-indigo-500 animate-pulse text-primary animate-bounce" />
              </div>
              <p className="text-sm font-medium">Thinking...</p>
            </div>
          ) : error ? (
            <div className="h-[400px] flex flex-col items-center justify-center gap-4 text-center p-8 bg-destructive/5 rounded-2xl border border-destructive/20">
              <AlertCircle className="h-10 w-10 text-destructive" />
              <div className="space-y-1">
                <p className="font-semibold text-destructive">Processing Error</p>
                <p className="text-sm text-destructive/80 max-w-sm">{error}</p>
              </div>
              <Button
                variant="outline"
                onClick={startProcessing}
                className="mt-2 border-destructive/20 text-destructive hover:bg-destructive/5"
              >
                Try Again
              </Button>
            </div>
          ) : proposedTransactions.length === 0 ? (
            <div className="h-[400px] flex flex-col items-center justify-center gap-4 text-center border-2 border-dashed rounded-2xl bg-muted/20">
              <div className="h-12 w-12 rounded-full bg-emerald-100 flex items-center justify-center">
                <Check className="h-6 w-6 text-emerald-600" />
              </div>
              <p className="text-sm font-semibold">Everything looks clear!</p>
              <p className="text-xs text-muted-foreground max-w-xs">
                No pending notifications were found in your inbox.
              </p>
            </div>
          ) : (
            <div className="border rounded-2xl bg-card overflow-hidden shadow-sm">
              <div className="max-h-[500px] overflow-auto">
                <table className="w-full text-sm border-collapse">
                  <thead className="bg-muted/50 sticky top-0 z-10 backdrop-blur-sm border-b">
                    <tr>
                      <th className="w-12 p-4 text-center">
                        <input
                          type="checkbox"
                          className="w-4 h-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-600 cursor-pointer"
                          checked={
                            approvedIndices.size === proposedTransactions.length && proposedTransactions.length > 0
                          }
                          onChange={toggleAll}
                          disabled={step === 'saving'}
                        />
                      </th>
                      <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Details
                      </th>
                      <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Category
                      </th>
                      <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Memo
                      </th>
                      <th className="text-left p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Tag
                      </th>
                      <th className="text-right p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Amount
                      </th>
                      <th className="text-center p-4 font-semibold text-[10px] uppercase tracking-wider text-muted-foreground">
                        Status
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {proposedTransactions.map((tx, i) => (
                      <tr
                        key={i}
                        className={`group border-b last:border-0 transition-all cursor-pointer hover:bg-muted/30 ${
                          !approvedIndices.has(i) && step !== 'saving' ? 'bg-muted/10' : ''
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
                            <span className="font-bold text-[10px] text-indigo-600 mb-0.5 uppercase tracking-wide">
                              {tx.accountName}
                            </span>
                            <span className="font-semibold text-sm leading-tight mb-1">{tx.payee}</span>
                            <span className="text-[10px] text-muted-foreground font-medium">
                              {new Date(tx.date).toLocaleDateString(undefined, {
                                day: '2-digit',
                                month: 'short',
                                year: 'numeric',
                              })}
                            </span>
                          </div>
                        </td>
                        <td className="p-4">
                          <CategoryBadge category={tx.category} type={tx.categoryType} />
                        </td>
                        <td className="p-4">
                          <span className="text-[10px] text-muted-foreground/50 italic">Empty</span>
                        </td>
                        <td className="p-4">
                          <span className="text-[10px] text-muted-foreground/50 italic">None</span>
                        </td>
                        <td className="p-4 text-right">
                          <div
                            className={`font-black text-base tabular-nums whitespace-nowrap ${tx.type === 'deposit' ? 'text-emerald-600' : 'text-foreground'}`}
                          >
                            {tx.accountName.toLowerCase().includes('binance') ? (
                              <TooltipProvider delayDuration={0}>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <span className="cursor-help underline decoration-dashed decoration-muted-foreground/50 underline-offset-4">
                                      {tx.type === 'deposit' ? '+' : '-'}
                                      <MaskedBalance amount={tx.amount * exchangeRate} />
                                    </span>
                                  </TooltipTrigger>
                                  <TooltipContent
                                    side="top"
                                    className="max-w-xs text-xs font-mono break-all font-medium py-2"
                                  >
                                    <div className="text-[10px] text-muted-foreground mb-1 font-sans font-semibold uppercase tracking-wider">
                                      Formula Preview
                                    </div>
                                    ={tx.amount} * IF(INDIRECT("A" & ROW())="{tx.accountName}";
                                    GOOGLEFINANCE("CURRENCY:USDVND"); 1)
                                    <div className="mt-2 text-[10px] text-muted-foreground font-sans">
                                      Current USD/VND Rate: <MaskedBalance amount={exchangeRate} />
                                    </div>
                                  </TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            ) : (
                              <>
                                {tx.type === 'deposit' ? '+' : '-'}
                                <MaskedBalance amount={tx.amount} />
                              </>
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
                              <span className="text-[9px] font-bold text-muted-foreground/60 uppercase tracking-widest group-hover:text-primary transition-colors">
                                Pending
                              </span>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>

        <DialogFooter className="p-6 bg-muted/30 border-t">
          <div className="flex items-center justify-between w-full">
            <div className="flex flex-col gap-0.5">
              <p className="text-xs font-semibold">
                {approvedIndices.size} of {proposedTransactions.length} items
              </p>
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-medium">
                Selected for import
              </p>
            </div>
            <div className="flex gap-3">
              <Button
                variant="ghost"
                onClick={() => onOpenChange(false)}
                disabled={step === 'saving'}
                className="text-xs"
              >
                Cancel
              </Button>
              <Button
                disabled={approvedIndices.size === 0 || step === 'saving'}
                onClick={handleSave}
                className="gap-2 shadow-lg shadow-primary/25 min-w-[160px] relative overflow-hidden h-10 transition-all active:scale-95"
              >
                {step === 'saving' ? (
                  <>
                    <Sparkles className="h-4 w-4 animate-pulse" />
                    Thinking...
                  </>
                ) : (
                  <>
                    <Sparkles className="h-4 w-4" />
                    Approve & Import
                  </>
                )}
              </Button>
            </div>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
