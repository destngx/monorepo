'use client';

import { useState, useEffect, useRef } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Sparkles, ShieldAlert } from 'lucide-react';
import { Loan } from '../model/types';
import ReactMarkdown from 'react-markdown';

interface Props {
  loans: Loan[];
}

export function LoanReviewAI({ loans }: Props) {
  const [loading, setLoading] = useState(false);
  const [review, setReview] = useState<string | null>(null);
  const hasGeneratedRef = useRef(false);

  const generateReview = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/ai/loan-review', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ loans }),
      });
      const data = await response.json();
      setReview(data.review);
    } catch (e) {
      console.error(e);
      setReview('Failed to generate AI review. Please try again later.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (!hasGeneratedRef.current && loans.length > 0) {
      hasGeneratedRef.current = true;
      void generateReview();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loans.length]);

  return (
    <Card className="border-orange-100 dark:border-orange-900/50 bg-orange-50/30 dark:bg-orange-950/10 shadow-sm overflow-hidden border-dashed">
      <CardHeader className="pb-2 pt-3 px-4 border-b border-orange-100/50 dark:border-orange-900/30">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-2">
            <ShieldAlert className="h-4 w-4 text-orange-500" />
            <CardTitle className="text-sm font-semibold text-orange-900 dark:text-orange-300">
              Lộc Phát Tài&apos;s Debt Strategy
            </CardTitle>
          </div>
          <div className="flex items-center gap-1.5">
            <Sparkles className="h-3 w-3 text-orange-400" />
            <span className="text-[10px] font-bold text-orange-600/60 dark:text-orange-400/60 uppercase tracking-widest">
              AI Strategy
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent className="p-4">
        {loading ? (
          <div className="flex items-center justify-center gap-3 py-8 text-xs text-muted-foreground animate-pulse">
            <Sparkles className="h-5 w-5 text-orange-400 animate-bounce" />
            <span>Thinking...</span>
          </div>
        ) : !review ? (
          <div className="flex flex-col items-center gap-3 py-6">
            <p className="text-xs text-muted-foreground text-center max-w-sm italic">
              Developing your debt settlement strategy...
            </p>
          </div>
        ) : (
          <div className="w-full">
            <div className="prose prose-sm dark:prose-invert max-w-none text-xs leading-relaxed text-slate-700 dark:text-slate-300 italic">
              <ReactMarkdown>{review}</ReactMarkdown>
            </div>
            <div className="flex justify-end gap-2 pt-3 mt-4 border-t border-orange-100/30 dark:border-orange-900/20">
              <Button
                variant="ghost"
                size="sm"
                onClick={generateReview}
                disabled={loading}
                className="text-[10px] h-7 px-3 gap-1.5 text-orange-600 dark:text-orange-400 hover:text-orange-700 hover:bg-orange-100/50 transition-all font-semibold"
              >
                <Sparkles className="h-3 w-3" />
                Update Strategy
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
