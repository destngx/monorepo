'use client';

import { X, Maximize2, Trash2, Sparkles, LayoutDashboard, Wallet, Target, CreditCard } from 'lucide-react';
import { Button } from '@wealth-management/ui';
import { cn } from '@wealth-management/utils';

interface Props {
  activeContext: string;
  onClose: () => void;
  isFullPage: boolean;
  setIsFullPage: (val: boolean) => void;
  onClearChat?: () => void;
  onRegenerateSuggestions?: () => void;
  isBusySuggestions: boolean;
  hasMessages: boolean;
}

const getContextIcon = (context: string) => {
  const ctx = context.toLowerCase();
  if (ctx.includes('budget')) return <Wallet className="h-3 w-3" />;
  if (ctx.includes('goal')) return <Target className="h-3 w-3" />;
  if (ctx.includes('card')) return <CreditCard className="h-3 w-3" />;
  return <LayoutDashboard className="h-3 w-3" />;
};

export function DrawerHeader({
  activeContext,
  onClose,
  isFullPage,
  setIsFullPage,
  onClearChat,
  onRegenerateSuggestions,
  isBusySuggestions,
  hasMessages,
}: Props) {
  return (
    <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/20">
      <div className="flex items-center gap-2">
        <div
          className={cn(
            'flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider',
            'bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400 border border-emerald-500/20',
          )}
        >
          <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
          {getContextIcon(activeContext)}
          {activeContext}
        </div>
        {onRegenerateSuggestions && !hasMessages && (
          <Button
            variant="ghost"
            size="sm"
            className="h-7 px-2 text-[10px] gap-1 text-muted-foreground hover:text-primary transition-colors"
            onClick={onRegenerateSuggestions}
            disabled={isBusySuggestions}
          >
            <Sparkles className={cn('h-3 w-3', isBusySuggestions && 'animate-spin')} />
            Suggest
          </Button>
        )}
      </div>
      <div className="flex items-center gap-1">
        {onClearChat && hasMessages && (
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground hover:text-destructive transition-colors"
            onClick={onClearChat}
            title="Reset Chat"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground"
          onClick={() => setIsFullPage(!isFullPage)}
        >
          <Maximize2 className={cn("h-4 w-4 transition-transform", isFullPage && "rotate-180")} />
        </Button>
        <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground" onClick={onClose}>
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
