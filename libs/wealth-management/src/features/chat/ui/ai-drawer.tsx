'use client';

import * as React from 'react';
import { Send, Sparkles } from 'lucide-react';
import { Sheet, SheetContent, SheetTitle, ScrollArea } from '@wealth-management/ui';
import { cn } from '@wealth-management/utils';
import { ChatMessage, SuggestionItem } from '../model/types';
import { MessageContent } from './components/MessageContent';
import { DrawerHeader } from './components/DrawerHeader';
import { SuggestionSection } from './components/SuggestionSection';

interface AIDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  messages: ChatMessage[];
  input: string;
  onInputChange: (val: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onPromptClick: (prompt: string) => void;
  onRegenerateSuggestions?: () => void;
  onClearChat?: () => void;
  isBusy: boolean;
  isBusySuggestions?: boolean;
  activeContext?: string;
  suggestedPrompts: SuggestionItem[];
  children?: React.ReactNode;
}

export function AIDrawer({
  isOpen,
  onClose,
  messages,
  input,
  onInputChange,
  onSubmit,
  onPromptClick,
  onRegenerateSuggestions,
  onClearChat,
  isBusy,
  isBusySuggestions = false,
  activeContext = 'Overview',
  suggestedPrompts,
  children,
}: AIDrawerProps) {
  const [isFullPage, setIsFullPage] = React.useState(false);
  const scrollRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isBusy]);

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent
        side="bottom"
        showCloseButton={false}
        className={cn(
          'p-0 flex flex-col transition-all duration-500 ease-in-out border-t shadow-2xl overflow-hidden',
          'md:bottom-6 md:right-6 md:left-auto md:fixed md:w-[400px] md:h-[600px] md:rounded-3xl md:border',
          isFullPage ? 'h-[94vh] md:h-[90vh] md:w-[600px]' : 'h-[65vh] md:h-[600px] md:w-[400px]',
        )}
      >
        <SheetTitle className="sr-only">AI Wealth Advisor Chat</SheetTitle>

        <DrawerHeader 
          activeContext={activeContext}
          onClose={onClose}
          isFullPage={isFullPage}
          setIsFullPage={setIsFullPage}
          onClearChat={onClearChat}
          onRegenerateSuggestions={onRegenerateSuggestions}
          isBusySuggestions={isBusySuggestions}
          hasMessages={messages.length > 0}
        />

        <div className="flex-1 flex flex-col min-h-0">
          <ScrollArea className="flex-1 p-4 overflow-scroll no-scrollbar" ref={scrollRef}>
            <div className="space-y-6 pb-4">
              {children}

              {messages.length === 0 && !children && (
                <SuggestionSection 
                  isBusySuggestions={isBusySuggestions}
                  activeContext={activeContext}
                  suggestedPrompts={suggestedPrompts}
                  onPromptClick={onPromptClick}
                />
              )}

              {messages.map((m, idx) => (
                <div key={idx} className={cn(
                  'flex flex-col gap-2 max-w-[85%]',
                  m.role === 'user' ? 'ml-auto items-end animate-in slide-in-from-right-2' : 'mr-auto items-start animate-in slide-in-from-left-2'
                )}>
                  <div className={cn(
                    'px-4 py-3 rounded-2xl text-sm leading-relaxed border',
                    m.role === 'user'
                      ? 'bg-primary text-primary-foreground rounded-tr-none border-primary shadow-sm'
                      : 'bg-muted text-foreground rounded-tl-none border-border/50 shadow-inner'
                  )}>
                    <MessageContent message={m} />
                  </div>
                </div>
              ))}

              {isBusy && (
                <div className="flex gap-2 max-w-[85%] mr-auto items-start font-medium italic text-muted-foreground">
                  <div className="bg-muted px-4 py-3 rounded-2xl rounded-tl-none border border-border/50 shadow-inner flex items-center gap-2">
                    <Sparkles className="h-4 w-4 animate-pulse text-indigo-500" />
                    <div className="flex gap-1.5 items-center">
                      <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce" />
                      <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce [animation-delay:0.2s]" />
                      <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce [animation-delay:0.4s]" />
                    </div>
                  </div>
                </div>
              )}
            </div>
          </ScrollArea>

          <div className="px-4 py-2 border-t bg-muted/40 backdrop-blur-sm">
            <div className="flex gap-2.5 overflow-x-auto pb-1.5 no-scrollbar scroll-smooth">
              {suggestedPrompts.map((p, i) => (
                <button
                  key={i}
                  onClick={() => onPromptClick(p.prompt)}
                  className="shrink-0 px-4 py-1.5 rounded-full border bg-background text-[11px] font-bold shadow-sm hover:border-primary/50 hover:bg-primary/5 active:scale-95 transition-all"
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          <form onSubmit={onSubmit} className="p-4 bg-background border-t">
            <div className="relative flex items-center group">
              <input
                value={input}
                onChange={(e) => onInputChange(e.target.value)}
                placeholder="Message AI Wealth Advisor..."
                className="w-full pl-4 pr-12 py-3.5 rounded-2xl border bg-muted/30 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all shadow-inner"
              />
              <button
                type="submit"
                disabled={!input.trim() || isBusy}
                className="absolute right-2.5 p-2 rounded-xl bg-primary text-primary-foreground disabled:opacity-50 transition-all hover:scale-105 active:scale-95 shadow-md"
              >
                <Send className="h-4 w-4" />
              </button>
            </div>
          </form>
        </div>
      </SheetContent>
    </Sheet>
  );
}
