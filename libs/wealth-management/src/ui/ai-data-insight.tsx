'use client';

import { useState, useCallback } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Sparkles, X, Loader2, RotateCcw } from 'lucide-react';
import { Button } from './button';
import { Popover as PopoverPrimitive, Portal as PortalPrimitive } from 'radix-ui';
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface AIDataInsightProps {
  type: string;
  description: string;
  data: any;
  market?: string;
  timeframe?: string;
}

export function AIDataInsight({ type, description, data, market, timeframe }: AIDataInsightProps) {
  const [insight, setInsight] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

  const fetchInsight = useCallback(async () => {
    // If we already have an insight, we just need the popover to open
    if (insight) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/ai/chart-insight', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ chartType: type, chartDescription: description, chartData: data, market, timeframe }),
      });

      if (!response.ok) throw new Error('Failed to fetch insight');

      const result = await response.json();
      setInsight(result.insight || 'No insight generated.');
    } catch (err: any) {
      const message = getErrorMessage(err);
      const errorMsg = isAppError(err) ? err.userMessage : message;
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [type, description, data, market, timeframe, insight]);

  const handleRefresh = useCallback(async () => {
    setInsight(null);
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/ai/chart-insight', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ chartType: type, chartDescription: description, chartData: data, market, timeframe }),
      });

      if (!response.ok) throw new Error('Failed to fetch insight');

      const result = await response.json();
      setInsight(result.insight || 'No insight generated.');
    } catch (err: any) {
      const message = getErrorMessage(err);
      const errorMsg = isAppError(err) ? err.userMessage : message;
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  }, [type, description, data, market, timeframe]);

  return (
    <PopoverPrimitive.Root
      open={open}
      onOpenChange={(val) => {
        setOpen(val);
        if (val) void fetchInsight();
      }}
    >
      <PopoverPrimitive.Trigger asChild>
        <Button
          variant="ghost"
          size="sm"
          className={`h-7 w-7 p-0 rounded-full transition-all duration-300 ${
            open
              ? 'bg-violet-500/20 text-violet-400 ring-1 ring-violet-500/30'
              : 'hover:bg-violet-500/10 text-zinc-400 hover:text-violet-400'
          }`}
          title="AI Chart Analysis"
          onClick={(e) => {
            e.stopPropagation();
          }}
        >
          {loading && !insight ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Sparkles className="h-3.5 w-3.5" />
          )}
        </Button>
      </PopoverPrimitive.Trigger>

      <PortalPrimitive.Root>
        <PopoverPrimitive.Content
          side="top"
          align="end"
          sideOffset={8}
          className="z-[9999] w-[340px] max-w-[90vw] animate-in fade-in zoom-in-95 duration-200"
        >
          <div className="rounded-xl border border-violet-500/30 bg-white dark:bg-zinc-950 shadow-2xl ring-1 ring-black/5 overflow-hidden">
            {/* Header */}
            <div className="flex items-center justify-between px-3 py-2 border-b border-zinc-200 dark:border-zinc-800/50 bg-violet-50 dark:bg-violet-950/40">
              <div className="flex items-center gap-1.5">
                <Sparkles className="h-3 w-3 text-violet-500" />
                <span className="text-[10px] font-bold uppercase tracking-widest text-violet-600 dark:text-violet-400">
                  AI Insight
                </span>
                <span className="text-[9px] text-zinc-400 font-mono truncate max-w-[120px]">· {type}</span>
              </div>
              <div className="flex items-center gap-1">
                {insight && (
                  <Button
                    onClick={handleRefresh}
                    variant="ghost"
                    size="sm"
                    className="h-5 w-5 p-0 text-zinc-400 hover:text-violet-400"
                    title="Refresh analysis"
                    disabled={loading}
                  >
                    <RotateCcw className={`h-2.5 w-2.5 ${loading ? 'animate-spin' : ''}`} />
                  </Button>
                )}
                <Button
                  onClick={() => setOpen(false)}
                  variant="ghost"
                  size="sm"
                  className="h-5 w-5 p-0 text-zinc-400 hover:text-zinc-500 dark:hover:text-zinc-200"
                >
                  <X className="h-3 w-3" />
                </Button>
              </div>
            </div>

            {/* Content */}
            <div className="px-3 py-3 max-h-[300px] overflow-y-auto bg-white dark:bg-zinc-950">
              {loading && !insight && (
                <div className="space-y-2 py-2">
                  <div className="h-3 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse w-full" />
                  <div className="h-3 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse w-4/5" />
                  <div className="h-3 bg-zinc-100 dark:bg-zinc-800 rounded animate-pulse w-3/5" />
                  <div className="flex items-center gap-2 mt-4 text-[10px] text-violet-500 font-mono font-bold uppercase tracking-tighter">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Synthesizing {type}...
                  </div>
                </div>
              )}

              {error && (
                <div className="text-[11px] text-rose-500 font-mono py-2 flex items-center gap-2">
                  <span className="p-1 rounded bg-rose-500/10">⚠</span> {error}
                </div>
              )}

              {insight && (
                <div className="prose prose-xs dark:prose-invert max-w-none text-[11px] leading-relaxed prose-p:my-1 prose-headings:my-1.5 prose-headings:text-[12px] prose-strong:text-violet-600 dark:prose-strong:text-violet-400 prose-ul:my-1 prose-li:my-0">
                  <ReactMarkdown remarkPlugins={[remarkGfm]}>{insight}</ReactMarkdown>
                </div>
              )}
            </div>
          </div>
        </PopoverPrimitive.Content>
      </PortalPrimitive.Root>
    </PopoverPrimitive.Root>
  );
}
