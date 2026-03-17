"use client";

import * as React from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { 
  X, 
  Maximize2, 
  ChevronRight, 
  Send, 
  Sparkles, 
  LayoutDashboard,
  Target,
  Wallet,
  CreditCard,
  Zap,
  CheckCircle,
  Trash2
} from "lucide-react";
import { 
  Sheet, 
  SheetContent, 
  SheetTitle,
} from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { cn } from "@wealth-management/utils";

interface AIDrawerProps {
  isOpen: boolean;
  onClose: () => void;
  messages: any[];
  input: string;
  onInputChange: (val: string) => void;
  onSubmit: (e: React.FormEvent) => void;
  onPromptClick: (prompt: string) => void;
  onRegenerateSuggestions?: () => void;
  onClearChat?: () => void;
  isBusy: boolean;
  isBusySuggestions?: boolean;
  activeContext?: string;
  suggestedPrompts: Array<{ label: string; prompt: string }>;
  children?: React.ReactNode; // For Insight Cards
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
  activeContext = "Overview",
  suggestedPrompts,
  children
}: AIDrawerProps) {
  const [isFullPage, setIsFullPage] = React.useState(false);
  const scrollRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight;
    }
  }, [messages, isBusy]);

  const getContextIcon = (context: string) => {
    const ctx = context.toLowerCase();
    if (ctx.includes("budget")) return <Wallet className="h-3 w-3" />;
    if (ctx.includes("goal")) return <Target className="h-3 w-3" />;
    if (ctx.includes("card")) return <CreditCard className="h-3 w-3" />;
    return <LayoutDashboard className="h-3 w-3" />;
  };

  const getTextContent = (message: any): string => {
    if (Array.isArray(message.parts)) {
      return message.parts
        .filter((p: any) => p.type === "text" && !!p.text)
        .map((p: any) => p.text)
        .join("");
    }
    if (message.content && typeof message.content === 'string') return message.content;
    return "";
  };

  const renderMessageContent = (message: any) => {
    const parts = message.parts || [];
    const toolInvocations = message.toolInvocations || [];
    const content = getTextContent(message);

    return (
      <div className="space-y-3">
        {content && !parts.some((p: any) => p.type === 'text') && (
          <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
          </div>
        )}
        
        {parts.map((part: any, idx: number) => {
          if (part.type === 'text' && part.text) {
            return (
              <div key={`text-${idx}`} className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 border-t border-border/10 pt-2 first:pt-0 first:border-0">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{part.text}</ReactMarkdown>
              </div>
            );
          }
          if (typeof part.type === 'string' && part.type.startsWith('tool-')) {
            const toolName = part.type.split('-')[1];
            const isDone = part.state === 'output-available';
            return (
               <div key={`tool-${idx}`} className="flex items-center gap-1.5 text-[10px] bg-muted/50 px-2 py-1.5 rounded border border-border/50 font-medium">
                 {isDone ? <CheckCircle className="h-3 w-3 text-emerald-500" /> : <Zap className="h-3 w-3 text-amber-500 animate-pulse" />}
                 {isDone ? `Finished: ${toolName}` : `Running: ${toolName}...`}
               </div>
            );
          }
          return null;
        })}

        {toolInvocations.map((tool: any, idx: number) => (
          <div key={`legacy-tool-${idx}`} className="flex items-center gap-1.5 text-[10px] bg-muted/50 px-2 py-1.5 rounded border border-border/50 font-medium">
            {tool.state === 'result' ? <CheckCircle className="h-3 w-3 text-emerald-500" /> : <Zap className="h-3 w-3 text-amber-500 animate-pulse" />}
            {tool.state === 'result' ? `Finished: ${tool.toolName}` : `Running: ${tool.toolName}...`}
          </div>
        ))}
      </div>
    );
  };

  return (
    <Sheet open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <SheetContent 
        side="bottom" 
        showCloseButton={false}
        className={cn(
          "p-0 flex flex-col transition-all duration-500 ease-in-out border-t shadow-2xl overflow-hidden",
          "md:bottom-6 md:right-6 md:left-auto md:fixed md:w-[400px] md:h-[600px] md:rounded-3xl md:border",
          isFullPage ? "h-[94vh] md:h-[90vh] md:w-[600px]" : "h-[65vh] md:h-[600px] md:w-[400px]"
        )}
      >
        <SheetTitle className="sr-only">AI Wealth Advisor Chat</SheetTitle>

        <div className="flex items-center justify-between px-4 py-2 border-b">
          <div className="flex items-center gap-2">
            <div className={cn(
               "flex items-center gap-1.5 px-2 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider",
               "bg-emerald-500/10 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-400 border border-emerald-500/20"
            )}>
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-500 animate-pulse" />
              {getContextIcon(activeContext)}
              {activeContext}
            </div>
            {onRegenerateSuggestions && messages.length === 0 && (
              <Button 
                variant="ghost" 
                size="sm" 
                className="h-7 px-2 text-[10px] gap-1 text-muted-foreground hover:text-primary transition-colors"
                onClick={onRegenerateSuggestions}
                disabled={isBusySuggestions}
              >
                <Sparkles className={cn("h-3 w-3", isBusySuggestions && "animate-spin")} />
                Suggest Questions
              </Button>
            )}
          </div>
          <div className="flex items-center gap-1">
            {onClearChat && messages.length > 0 && (
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
            <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground" onClick={() => setIsFullPage(!isFullPage)}>
              <Maximize2 className="h-4 w-4" />
            </Button>
            <Button variant="ghost" size="icon" className="h-8 w-8 text-muted-foreground" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        <div className="flex-1 flex flex-col min-h-0">
          <ScrollArea className="flex-1 p-4 overflow-scroll no-scrollbar" ref={scrollRef}>
            <div className="space-y-6 pb-4">
              {children}
              
              {messages.length === 0 && !children && (
                <div className="flex flex-col items-center justify-center py-10 text-center space-y-6">
                  <div className="h-12 w-12 rounded-2xl bg-primary/10 flex items-center justify-center text-primary">
                    <Sparkles className={cn("h-6 w-6", isBusySuggestions && "animate-spin")} />
                  </div>
                  <div className="space-y-2">
                    <h3 className="font-bold text-lg leading-tight text-foreground">
                      {isBusySuggestions ? "Scanning your finances..." : `Suggestions for ${activeContext}`}
                    </h3>
                    <p className="text-sm text-muted-foreground max-w-[280px] mx-auto">
                      {isBusySuggestions 
                        ? "AI is analyzing the page to generate the best questions for you." 
                        : "Based on your current view, you might want to ask:"}
                    </p>
                  </div>
                  
                  <div className="grid grid-cols-1 gap-2.5 w-full max-w-sm px-4">
                    {isBusySuggestions ? (
                      Array.from({ length: 3 }).map((_, i) => (
                        <div key={i} className="h-14 w-full rounded-2xl bg-muted animate-pulse" />
                      ))
                    ) : (
                      suggestedPrompts.map((p, i) => (
                        <button
                          key={i}
                          onClick={() => onPromptClick(p.prompt)}
                          className="flex items-center justify-between p-3.5 rounded-2xl border bg-card/50 backdrop-blur-sm hover:border-primary/50 hover:bg-primary/5 active:scale-[0.98] transition-all text-sm font-semibold text-left group"
                        >
                          <span className="flex-1 pr-4 leading-normal">{p.prompt}</span>
                          <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors shrink-0" />
                        </button>
                      ))
                    )}
                  </div>
                </div>
              )}

              {messages.map((m, idx) => (
                <div key={idx} className={cn(
                  "flex flex-col gap-2 max-w-[85%]",
                  m.role === "user" ? "ml-auto items-end" : "mr-auto items-start"
                )}>
                  <div className={cn(
                    "px-4 py-3 rounded-2xl text-sm leading-relaxed",
                    m.role === "user" 
                      ? "bg-primary text-primary-foreground rounded-tr-none" 
                      : "bg-muted text-foreground rounded-tl-none border border-border/50"
                  )}>
                    {m.role === "user" ? getTextContent(m) : renderMessageContent(m)}
                  </div>
                </div>
              ))}

              {isBusy && (
                <div className="flex gap-2 max-w-[85%] mr-auto items-start">
                  <div className="bg-muted px-4 py-3 rounded-2xl rounded-tl-none border border-border/50">
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

          <div className="px-4 py-2 border-t bg-muted/30">
            <div className="flex gap-2 overflow-x-auto pb-1 no-scrollbar">
              {suggestedPrompts.map((p, i) => (
                <button
                  key={i}
                  onClick={() => onPromptClick(p.prompt)}
                  className="shrink-0 px-3 py-1.5 rounded-full border bg-background text-[11px] font-semibold hover:border-primary/50 transition-colors"
                >
                  {p.label}
                </button>
              ))}
            </div>
          </div>

          <form onSubmit={onSubmit} className="p-4 bg-background border-t">
            <div className="relative flex items-center">
              <input
                value={input}
                onChange={(e) => onInputChange(e.target.value)}
                placeholder="Message AI Wealth Advisor..."
                className="w-full pl-4 pr-12 py-3 rounded-2xl border bg-muted/30 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all"
              />
              <button
                type="submit"
                disabled={!input.trim() || isBusy}
                className="absolute right-2 p-2 rounded-xl bg-primary text-primary-foreground disabled:opacity-50 transition-all hover:scale-105"
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
