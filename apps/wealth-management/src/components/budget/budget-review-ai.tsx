"use client";

import { useState, useEffect } from "react";
import { Sparkles, RefreshCw } from "lucide-react";
import ReactMarkdown from "react-markdown";
import { Button } from "@/components/ui/button";

interface BudgetReviewAIProps {
  budget: any[];
  transactions: any[];
  totalSpent: number;
  totalLimit: number;
  view: string;
  date: string;
}

export function BudgetReviewAI({ budget, transactions, totalSpent, totalLimit, view, date }: BudgetReviewAIProps) {
  const [review, setReview] = useState<string>("");
  const [loading, setLoading] = useState(false);

  const fetchReview = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/ai/budget-review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          budget,
          transactions,
          totalSpent,
          totalLimit,
          view,
          date
        })
      });
      const data = await res.json();
      setReview(data.review || "Unable to generate review at this time.");
    } catch (err) {
      console.error("Budget Review AI Error:", err);
      setReview("Error loading financial review.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchReview();
  }, [view, date]);

  return (
    <div className="bg-indigo-50/50 dark:bg-indigo-950/20 rounded-xl p-4 border border-indigo-100/50 dark:border-indigo-900/30 relative overflow-hidden group">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-indigo-500/10 rounded-lg">
            <Sparkles className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
          </div>
          <div>
            <h3 className="text-sm font-bold text-indigo-900 dark:text-indigo-100 uppercase tracking-tight">
              Lộc Phát Tài Insights
            </h3>
            <p className="text-[10px] text-indigo-600/70 dark:text-indigo-400/70 font-medium">
              Real-time Budget Analysis • {view} review
            </p>
          </div>
        </div>
        <Button 
          variant="ghost" 
          size="icon" 
          onClick={fetchReview} 
          disabled={loading}
          className="h-8 w-8 text-indigo-600 hover:text-indigo-700 hover:bg-indigo-100/50 dark:text-indigo-400 dark:hover:text-indigo-300 dark:hover:bg-indigo-900/30"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${loading ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      <div className="relative min-h-[80px]">
        {loading ? (
          <div className="flex flex-col items-center justify-center py-8 space-y-3">
            <div className="relative">
              <Sparkles className="h-6 w-6 text-indigo-400 animate-pulse" />
              <div className="absolute inset-0 bg-indigo-400/20 blur-xl animate-pulse rounded-full" />
            </div>
            <p className="text-xs font-medium text-indigo-600/70 dark:text-indigo-400/70 animate-pulse">
              Thinking...
            </p>
          </div>
        ) : (
          <div className="prose prose-sm dark:prose-invert max-w-none text-indigo-900/80 dark:text-indigo-100/80 text-sm leading-relaxed italic pr-4">
            <ReactMarkdown>
              {review || "Analyzing your spending patterns to provide actionable insights..."}
            </ReactMarkdown>
          </div>
        )}
      </div>

      {/* Decorative background element */}
      <div className="absolute top-0 right-0 -mt-4 -mr-4 h-24 w-24 bg-indigo-500/5 blur-[60px] rounded-full pointer-events-none" />
    </div>
  );
}
