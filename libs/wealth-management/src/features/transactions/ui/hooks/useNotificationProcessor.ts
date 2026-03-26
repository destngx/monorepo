import { useState, useEffect } from 'react';
import { EmailNotification } from '@wealth-management/types';

export interface ProposedTransaction {
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

export function useNotificationProcessor(open: boolean, onOpenChange: (open: boolean) => void, onComplete: () => void) {
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

      fetch('/api/exchange-rate')
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
        setError(parsedTransactions.error || 'Failed to parse notifications.');
        setStep('review');
      }
    } catch (error: any) {
      setError(error.message);
      setStep('review');
    }
  };

  useEffect(() => {
    if (open) startProcessing();
    else reset();
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
        }
      }

      onComplete();
      setTimeout(() => onOpenChange(false), 1000);
    } catch (err: any) {
      setError(err.message);
      setStep('review');
    }
  };

  return {
    step,
    notifications,
    proposedTransactions,
    approvedIndices,
    error,
    exchangeRate,
    itemStatus,
    startProcessing,
    toggleApproval,
    toggleAll,
    handleSave,
  };
}
