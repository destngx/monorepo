'use client';

import { useState, useEffect } from 'react';
import { Sparkles, RefreshCw, Calendar } from 'lucide-react';
import ReactMarkdown from 'react-markdown';
import { Button } from "@wealth-management/ui/button";
import { Transaction } from '@wealth-management/types';
import { AIInsightRenderer } from '@wealth-management/ui';
import type { StructuredInsight } from '@wealth-management/ai/server';
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface TransactionReviewAIProps {
  transactions: Transaction[];
}

export function TransactionReviewAI({ transactions }: TransactionReviewAIProps) {
  const [review, setReview] = useState<StructuredInsight | string>('');
  const [loading, setLoading] = useState(false);

  const fetchReview = async () => {
    if (transactions.length === 0) return;
    setLoading(true);
    try {
      const res = await fetch('/api/ai/transaction-review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ transactions }),
      });
      const data = await res.json();
      setReview(data.review || 'Unable to generate review at this time.');
    } catch (err) {
      const message = getErrorMessage(err);
      const errorMsg = isAppError(err) ? err.userMessage : 'Error loading weekly insights.';
      console.error('Transaction Review AI Error:', message);
      setReview(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void fetchReview();
  }, [transactions.length]);

  const isStructured = typeof review === 'object' && review !== null && 'sections' in review;

  return (
    <div className="bg-amber-50/50 dark:bg-amber-950/10 rounded-xl p-4 border border-amber-100/50 dark:border-amber-900/20 relative overflow-hidden group">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-amber-500/10 rounded-lg">
            <Calendar className="h-4 w-4 text-amber-600 dark:text-amber-400" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-amber-900 dark:text-amber-100 uppercase tracking-tight flex items-center gap-1.5">
              Lộc Phát Tài Insights
              <span className="text-[10px] font-medium bg-amber-500/10 text-amber-600 px-1.5 py-0.5 rounded-full lowercase">
                weekly
              </span>
            </h3>
            <p className="text-[10px] text-amber-600/70 dark:text-amber-400/70 font-medium">Last 7 Days Analysis</p>
          </div>
        </div>
        <Button
          variant="ghost"
          size="icon"
          onClick={fetchReview}
          disabled={loading}
          className="h-8 w-8 text-amber-600 hover:text-amber-700 hover:bg-amber-100/50 dark:text-amber-400 dark:hover:text-amber-300 dark:hover:bg-amber-900/20"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <div className="relative min-h-[60px]">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-6 space-y-3 font-medium">
            <Sparkles className="h-5 w-5 text-amber-400 animate-pulse" />
            <p className="text-xs text-amber-600/70 dark:text-amber-400/70">Thinking...</p>
          </div>
        ) : isStructured ? (
          <AIInsightRenderer insight={review} />
        ) : (
          <div className="prose prose-sm dark:prose-invert max-w-none text-amber-900/80 dark:text-amber-100/80 text-sm leading-relaxed italic pr-4">
            <ReactMarkdown>{(review as string) || 'Analyzing your recent transaction history...'}</ReactMarkdown>
          </div>
        )}
      </div>

      {/* Decorative background element */}
      <div className="absolute top-0 right-0 -mt-4 -mr-4 h-20 w-20 bg-amber-500/5 blur-[40px] rounded-full pointer-events-none" />
    </div>
  );
}
