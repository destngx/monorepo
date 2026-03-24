'use client';

import { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import { Plus, ChevronUp, Loader2 } from 'lucide-react';
import { TransactionFilters } from '@/components/transactions/transaction-filters';
import { TransactionTable } from '@/components/transactions/transaction-table';
import { TransactionForm } from '@/components/transactions/transaction-form';
import { NotificationProcessor } from '@/components/transactions/notification-processor';
import { TransactionInput } from '@wealth-management/schemas';
import { Transaction } from '../model/types';
import { Loading } from '@/components/ui/loading';
import { format, subDays, isAfter } from 'date-fns';
import { useApiErrorHandler } from '../../../hooks/use-api-error-handler';
import { TransactionReviewAI } from '@/components/transactions/transaction-review-ai';
import { AIDataInsight } from '@/components/dashboard/ai-data-insight';
import { Sparkles } from 'lucide-react';

import { GoogleSheetsAlert } from '@/components/google-sheets/google-sheets-alert';

const PAGE_SIZE = 500;

function filterTransactions(txns: Transaction[], q: string): Transaction[] {
  if (!q.trim()) return txns;
  const lower = q.toLowerCase();
  return txns.filter(
    (t) =>
      t.payee?.toLowerCase().includes(lower) ||
      t.accountName?.toLowerCase().includes(lower) ||
      t.category?.toLowerCase().includes(lower) ||
      t.memo?.toLowerCase().includes(lower) ||
      t.tags?.some((tag) => tag.toLowerCase().includes(lower)),
  );
}

export default function TransactionsPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [notifOpen, setNotifOpen] = useState(false);
  const [pendingNotifCount, setPendingNotifCount] = useState(0);
  const [allTransactions, setAllTransactions] = useState<Transaction[]>([]);
  const [search, setSearch] = useState('');
  const [visibleCount, setVisibleCount] = useState(PAGE_SIZE);
  const [loading, setLoading] = useState(true);
  const [showScrollTop, setShowScrollTop] = useState(false);
  const [authError, setAuthError] = useState<'MISSING_CREDENTIALS' | 'OAUTH_EXPIRED' | 'API_ERROR' | null>(null);
  const topRef = useRef<HTMLDivElement>(null);
  const { withErrorHandling } = useApiErrorHandler();

  const fetchTransactions = async () => {
    setLoading(true);
    setAuthError(null);
    try {
      const res = await withErrorHandling(() => fetch('/api/transactions'), 'Failed to load transactions');
      if (res && res.ok) {
        const data: Transaction[] = await res.json();
        data.sort((a, b) => new Date(b.date).getTime() - new Date(a.date).getTime());
        setAllTransactions(data);
        setVisibleCount(PAGE_SIZE);
      } else if (res && res.status === 401) {
        const errorData = await res.json();
        setAuthError(errorData.code || 'API_ERROR');
      }
    } catch (e) {
      console.error(e);
      setAuthError('API_ERROR');
    } finally {
      setLoading(false);
    }
  };

  const fetchPendingCount = async () => {
    try {
      const res = await withErrorHandling(() => fetch('/api/notifications'), 'Failed to load notification count');
      if (res && res.ok) {
        const data = await res.json();
        setPendingNotifCount(data.length);
      }
    } catch (e) {
      console.error('Failed to fetch notifications count', e);
    }
  };

  useEffect(() => {
    fetchTransactions();
    fetchPendingCount();
  }, [withErrorHandling]);

  // Reset pagination when search changes
  const handleSearch = useCallback((q: string) => {
    setSearch(q);
    setVisibleCount(PAGE_SIZE);
  }, []);

  // Show scroll-to-top when scrolled past 400px
  useEffect(() => {
    const onScroll = () => setShowScrollTop(window.scrollY > 400);
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  const scrollToTop = useCallback(() => {
    topRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  // Filter first, then paginate
  const filtered = useMemo(() => filterTransactions(allTransactions, search), [allTransactions, search]);
  const visibleTransactions = filtered.slice(0, visibleCount);
  const hasMore = visibleCount < filtered.length;
  const remaining = filtered.length - visibleCount;

  const handleLoadMore = () => setVisibleCount((prev) => prev + PAGE_SIZE);

  const handleCreate = async (data: TransactionInput) => {
    await fetch('/api/transactions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    setFormOpen(false);
    fetchTransactions();
  };

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      <div ref={topRef} />

      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h2 className="text-2xl font-bold tracking-tight">Transactions</h2>
            <AIDataInsight
              type="Transaction Data"
              description="Overview of recent transactions including payee, category, and amounts."
              data={visibleTransactions}
            />
          </div>
          <p className="text-muted-foreground text-sm">
            {loading
              ? 'Thinking…'
              : search
                ? `${filtered.length} result${filtered.length !== 1 ? 's' : ''} for "${search}" · showing ${visibleTransactions.length}`
                : `${allTransactions.length} total · showing ${visibleTransactions.length}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2 w-full sm:w-auto">
          <div className="relative">
            <Button
              variant="outline"
              onClick={() => setNotifOpen(true)}
              className="gap-2 cursor-pointer border-indigo-200 bg-indigo-50/30 text-indigo-700 hover:bg-indigo-50 hover:text-indigo-800"
            >
              <Sparkles className="h-4 w-4" />
              Import from Email
            </Button>
            {pendingNotifCount > 0 && (
              <span className="absolute -top-1.5 -right-1.5 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-[10px] font-bold text-white shadow-sm ring-2 ring-background">
                {pendingNotifCount}
              </span>
            )}
          </div>
          <Button onClick={() => setFormOpen(true)} className="gap-2 cursor-pointer shadow-sm">
            <Plus className="h-4 w-4" />
            Add Transaction
          </Button>
        </div>
      </div>

      {/* Google Sheets Auth Alerts */}
      {authError && <GoogleSheetsAlert errorType={authError} />}

      {/* AI Transaction Review for Last 7 Days */}
      {!loading && allTransactions.length > 0 && !authError && (
        <TransactionReviewAI
          transactions={allTransactions.filter((t) => isAfter(new Date(t.date), subDays(new Date(), 7)))}
        />
      )}

      <div className="rounded-xl border bg-card text-card-foreground shadow p-4 sm:p-6">
        <TransactionFilters search={search} onSearch={handleSearch} />

        {loading ? (
          <Loading message="Thinking..." />
        ) : (
          <>
            <TransactionTable transactions={visibleTransactions} />

            {hasMore && (
              <div className="flex flex-col items-center gap-1 pt-6 pb-2">
                <Button variant="outline" className="gap-2 cursor-pointer min-w-[220px]" onClick={handleLoadMore}>
                  <Loader2 className="h-4 w-4" />
                  Load {Math.min(PAGE_SIZE, remaining)} more
                  <span className="text-muted-foreground text-xs">({remaining} remaining)</span>
                </Button>
              </div>
            )}

            {!hasMore && filtered.length > PAGE_SIZE && (
              <p className="text-center text-xs text-muted-foreground pt-4">All {filtered.length} transactions shown</p>
            )}
          </>
        )}
      </div>

      <TransactionForm open={formOpen} onOpenChange={setFormOpen} onSubmit={handleCreate} />
      <NotificationProcessor open={notifOpen} onOpenChange={setNotifOpen} onComplete={fetchTransactions} />

      {showScrollTop && (
        <button
          onClick={scrollToTop}
          className="fixed bottom-6 right-6 z-50 flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-lg transition-all hover:scale-110 hover:shadow-xl cursor-pointer"
          aria-label="Scroll to top"
        >
          <ChevronUp className="h-5 w-5" />
        </button>
      )}
    </div>
  );
}
