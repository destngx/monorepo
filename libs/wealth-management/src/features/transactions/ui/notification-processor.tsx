"use client";

import { 
  Dialog, 
  DialogContent, 
  DialogHeader, 
  DialogTitle, 
  DialogDescription,
  DialogFooter,
  Button,
  MaskedBalance
} from "@wealth-management/ui";
import { Check, AlertCircle, Sparkles } from "lucide-react";
import { useNotificationProcessor } from "./hooks/useNotificationProcessor";
import { NotificationReviewTable } from "./components/NotificationReviewTable";

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onComplete: () => void;
}

export function NotificationProcessor({ open, onOpenChange, onComplete }: Props) {
  const {
    step,
    proposedTransactions,
    approvedIndices,
    error,
    exchangeRate,
    itemStatus,
    startProcessing,
    toggleApproval,
    toggleAll,
    handleSave
  } = useNotificationProcessor(open, onOpenChange, onComplete);

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
              <Sparkles className="h-10 w-10 text-indigo-500 animate-pulse animate-bounce" />
              <p className="text-sm font-medium">Thinking...</p>
            </div>
          ) : error ? (
            <div className="h-[400px] flex flex-col items-center justify-center gap-4 text-center p-8 bg-destructive/5 rounded-2xl border border-destructive/20">
              <AlertCircle className="h-10 w-10 text-destructive" />
              <div className="space-y-1">
                <p className="font-semibold text-destructive">Processing Error</p>
                <p className="text-sm text-destructive/80 max-w-sm">{error}</p>
              </div>
              <Button variant="outline" onClick={startProcessing} className="mt-2 border-destructive/20 text-destructive hover:bg-destructive/5">
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
            <NotificationReviewTable 
              proposedTransactions={proposedTransactions}
              approvedIndices={approvedIndices}
              itemStatus={itemStatus}
              step={step}
              exchangeRate={exchangeRate}
              toggleApproval={toggleApproval}
              toggleAll={toggleAll}
            />
          )}
        </div>

        <DialogFooter className="p-6 bg-muted/30 border-t">
          <div className="flex items-center justify-between w-full">
            <div className="flex flex-col gap-0.5">
              <p className="text-xs font-semibold">
                {approvedIndices.size} of {proposedTransactions.length} items
              </p>
              <p className="text-[10px] text-muted-foreground uppercase tracking-wider font-medium">Selected for import</p>
            </div>
            <div className="flex gap-3">
              <Button variant="ghost" onClick={() => onOpenChange(false)} disabled={step === 'saving'} className="text-xs">
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
