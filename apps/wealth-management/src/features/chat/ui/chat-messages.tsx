'use client';

import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Bot, User, Zap } from 'lucide-react';
import { ScrollArea } from '@wealth-management/ui/scroll-area';

import { ChatMessage } from '../model/types';

interface ChatMessagesProps {
  messages: ChatMessage[];
  isLoading?: boolean;
}

export function ChatMessages({ messages, isLoading }: ChatMessagesProps) {
  const scrollRef = React.useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when messages change
  React.useEffect(() => {
    scrollRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isLoading]);

  if (messages.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center min-h-[400px]">
        <div className="text-center space-y-4">
          <div className="flex justify-center">
            <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center">
              <Bot className="h-6 w-6 text-primary" />
            </div>
          </div>
          <div>
            <h3 className="text-lg font-semibold">How can I help?</h3>
            <p className="text-sm text-muted-foreground mt-1 max-w-sm mx-auto">
              Ask about your finances, budget, net worth, or get personalized advice based on your data.
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <ScrollArea className="flex-1 p-6">
      <div className="space-y-6 max-w-3xl mx-auto">
        {messages.map((message) => (
          <div key={message.id} className={`flex gap-4 ${message.role === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>
            {/* Avatar */}
            <div
              className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg shadow-sm ${
                message.role === 'user' ? 'bg-primary text-primary-foreground' : 'bg-muted border'
              }`}
            >
              {message.role === 'user' ? <User className="h-4 w-4" /> : <Bot className="h-4 w-4" />}
            </div>

            {/* Message */}
            <div
              className={`flex flex-col gap-2 rounded-lg px-4 py-3 max-w-[70%] ${
                message.role === 'user'
                  ? 'bg-primary text-primary-foreground rounded-tr-none'
                  : 'bg-muted border rounded-tl-none'
              }`}
            >
              <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 prose-headings:my-2 prose-headings:text-sm prose-headings:font-semibold prose-ul:my-2 prose-ol:my-2 prose-li:my-0 prose-code:text-xs">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
              </div>
            </div>
          </div>
        ))}

        {/* Loading indicator */}
        {isLoading && (
          <div className="flex gap-4 flex-row">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg border bg-muted animate-pulse">
              <Zap className="h-4 w-4 animate-pulse text-indigo-500" />
            </div>
            <div className="bg-muted border rounded-lg rounded-tl-none px-4 py-3">
              <div className="flex gap-1.5 items-center">
                <span className="h-2 w-2 rounded-full bg-muted-foreground/40 animate-bounce [animation-delay:-0.3s]" />
                <span className="h-2 w-2 rounded-full bg-muted-foreground/40 animate-bounce [animation-delay:-0.15s]" />
                <span className="h-2 w-2 rounded-full bg-muted-foreground/40 animate-bounce" />
              </div>
            </div>
          </div>
        )}

        {/* Scroll anchor */}
        <div ref={scrollRef} />
      </div>
    </ScrollArea>
  );
}
