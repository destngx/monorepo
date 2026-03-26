'use client';

import { Sparkles, ChevronRight } from 'lucide-react';
import { cn } from '@wealth-management/utils';
import { SuggestionItem } from '../../model/types';

interface Props {
  isBusySuggestions: boolean;
  activeContext: string;
  suggestedPrompts: SuggestionItem[];
  onPromptClick: (prompt: string) => void;
}

export function SuggestionSection({
  isBusySuggestions,
  activeContext,
  suggestedPrompts,
  onPromptClick,
}: Props) {
  if (suggestedPrompts.length === 0 && !isBusySuggestions) return null;

  return (
    <div className="flex flex-col items-center justify-center py-10 text-center space-y-6 animate-in fade-in slide-in-from-bottom-2 duration-300">
      <div className="h-14 w-14 rounded-2xl bg-primary/10 flex items-center justify-center text-primary shadow-inner">
        <Sparkles className={cn('h-7 w-7', isBusySuggestions && 'animate-spin')} />
      </div>
      <div className="space-y-2 px-6">
        <h3 className="font-bold text-lg leading-tight text-foreground">
          {isBusySuggestions ? 'Scanning your finances...' : `Suggestions for ${activeContext}`}
        </h3>
        <p className="text-sm text-muted-foreground max-w-[280px] mx-auto">
          {isBusySuggestions
            ? 'AI is analyzing the page to generate the best questions for you.'
            : 'Based on your current view, you might want to ask:'}
        </p>
      </div>

      <div className="grid grid-cols-1 gap-2.5 w-full max-w-sm px-6">
        {isBusySuggestions
          ? Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-14 w-full rounded-2xl bg-muted animate-pulse border border-border/50" />
            ))
          : suggestedPrompts.map((p, i) => (
              <button
                key={i}
                onClick={() => onPromptClick(p.prompt)}
                className="flex items-center justify-between p-4 rounded-2xl border bg-card/50 backdrop-blur-sm hover:border-primary/50 hover:bg-primary/5 active:scale-[0.98] transition-all text-sm font-semibold text-left group"
              >
                <span className="flex-1 pr-4 leading-normal">{p.prompt}</span>
                <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors shrink-0" />
              </button>
            ))}
      </div>
    </div>
  );
}
