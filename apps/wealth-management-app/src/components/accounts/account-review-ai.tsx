"use client";

import { useState, useEffect, useRef } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Sparkles } from "lucide-react";
import { Account } from "@wealth-management/types";
import ReactMarkdown from "react-markdown";

interface Props {
  accounts: Account[];
  totalAssets: number;
  totalLiabilities: number;
  totalNetWorth: number;
}

export function AccountReviewAI({ accounts, totalAssets, totalLiabilities, totalNetWorth }: Props) {
  const [loading, setLoading] = useState(false);
  const [review, setReview] = useState<string | null>(null);
  const hasGeneratedRef = useRef(false);

  const generateReview = async () => {
    setLoading(true);
    try {
      const response = await fetch("/api/ai/account-review", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ accounts, totalAssets, totalLiabilities, totalNetWorth }),
      });
      const data = await response.json();
      setReview(data.review);
    } catch (e) {
      console.error(e);
      setReview("Failed to generate AI review. Please try again later.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!hasGeneratedRef.current && accounts.length > 0) {
      hasGeneratedRef.current = true;
      generateReview();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [accounts.length]);

  return (
    <Card className="border-indigo-100 dark:border-indigo-900/50 bg-indigo-50/30 dark:bg-indigo-950/10 shadow-sm overflow-hidden border-dashed">
      <CardHeader className="pb-2 pt-3 px-4 border-b border-indigo-100/50 dark:border-indigo-900/30">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2">
             <Sparkles className="h-4 w-4 text-indigo-500" />
             <CardTitle className="text-sm font-semibold text-indigo-900 dark:text-indigo-300">Lộc Phát Tài&apos;s Wealth Review</CardTitle>
          </div>
          <span className="text-[10px] font-bold text-indigo-600/60 dark:text-indigo-400/60 uppercase tracking-widest">AI Advisory</span>
        </div>
      </CardHeader>
      <CardContent className="p-4">
        {loading ? (
          <div className="flex items-center justify-center gap-3 py-8 text-xs text-muted-foreground animate-pulse">
            <Sparkles className="h-5 w-5 text-indigo-400 animate-bounce" />
            <span>Thinking...</span>
          </div>
        ) : !review ? (
          <div className="flex flex-col items-center gap-3 py-6">
            <p className="text-xs text-muted-foreground text-center max-w-sm italic">
              Analyzing your wealth structure and allocation...
            </p>
          </div>
        ) : (
          <div className="w-full">
            <div className="prose prose-sm dark:prose-invert max-w-none text-xs leading-relaxed text-slate-700 dark:text-slate-300 italic">
              <ReactMarkdown>{review}</ReactMarkdown>
            </div>
            <div className="flex justify-end gap-2 pt-3 mt-4 border-t border-indigo-100/30 dark:border-indigo-900/20">
              <Button 
                variant="ghost" 
                size="sm" 
                onClick={generateReview} 
                disabled={loading} 
                className="text-[10px] h-7 px-3 gap-1.5 text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 hover:bg-indigo-100/50 transition-all font-semibold"
              >
                <Sparkles className="h-3 w-3" />
                Refresh Analysis
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
