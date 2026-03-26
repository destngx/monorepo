'use client';

import { Bot, Sparkles } from 'lucide-react';

interface Props {
  mounted: boolean;
  activeModelLabel: string;
}

export function ChatHeader({ mounted, activeModelLabel }: Props) {
  return (
    <div className="flex items-center justify-between px-6 py-4 border-b bg-muted/30">
      <div className="flex items-center gap-3">
        <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center border border-primary/20">
          <Bot className="h-5 w-5 text-primary" />
        </div>
        <div>
          <h3 className="text-sm font-semibold">WealthOS AI Advisor</h3>
          <p className="text-[10px] text-muted-foreground flex items-center gap-1">
            <Sparkles className="h-2.5 w-2.5 text-amber-500" />
            {mounted ? activeModelLabel : 'Scanning your accounts...'}
          </p>
        </div>
      </div>
    </div>
  );
}
