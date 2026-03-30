'use client';

import { useState, useEffect, memo } from 'react';
import { useChat } from '@ai-sdk/react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Button } from "@wealth-management/ui/button";
import { ScrollArea } from '@wealth-management/ui/scroll-area';
import { Send, Bot, User, Sparkles, Zap, CheckCircle } from 'lucide-react';
import { useAISettings } from '@/hooks/use-ai-settings';
import { AI_MODELS } from '@wealth-management/ai';
import { useDebouncedChatPersistence } from '@/hooks/use-debounced-chat-persistence';

export function hasContent(message: any): boolean {
  if (Array.isArray(message.parts) && message.parts.length > 0) return true;
  if (message.toolInvocations && message.toolInvocations.length > 0) return true;
  if (message.content) return true;
  return message.role === 'user';
}

// Memoized message content component to prevent expensive re-renders
const MemoizedMessageContent = memo(({ message }: { message: any }) => {
  const parts = message.parts || [];
  const toolInvocations = message.toolInvocations || [];

  if (parts.length === 0 && toolInvocations.length === 0 && !message.content) return null;

  return (
    <div className="space-y-4">
      {/* 1. Legacy Content */}
      {message.content && parts.length === 0 && (
        <div className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 prose-headings:my-2 prose-headings:font-semibold prose-ul:my-2 prose-ol:my-2 prose-li:my-0">
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.content}</ReactMarkdown>
        </div>
      )}

      {/* 2. Parts */}
      {parts.map((part: any, idx: number) => {
        if (part.type === 'text' && part.text) {
          return (
            <div
              key={`text-${idx}`}
              className="prose prose-sm dark:prose-invert max-w-none prose-p:my-0 prose-p:first:mt-0 prose-headings:my-2 prose-headings:font-semibold prose-ul:my-2 prose-ol:my-2 prose-li:my-0"
            >
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{part.text}</ReactMarkdown>
            </div>
          );
        }

        if (typeof part.type === 'string' && part.type.startsWith('tool-')) {
          const toolName = part.type.split('-')[1];
          if (
            part.state === 'input-streaming' ||
            part.state === 'input-available' ||
            part.state === 'approval-requested'
          ) {
            return (
              <div
                key={`tool-call-${idx}`}
                className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded px-3 py-2 text-xs"
              >
                <p className="font-semibold text-blue-700 dark:text-blue-300 flex items-center gap-1">
                  <Zap className="h-3 w-3" /> Calling: {toolName}
                </p>
              </div>
            );
          }
          if (part.state === 'output-available') {
            return (
              <div
                key={`tool-result-${idx}`}
                className="bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded px-3 py-2 text-xs"
              >
                <p className="font-semibold text-green-700 dark:text-green-300 flex items-center gap-1">
                  <CheckCircle className="h-3 w-3" /> {toolName}
                </p>
                {typeof part.output === 'string' ? (
                  <p className="mt-1 whitespace-pre-wrap">{part.output}</p>
                ) : (
                  <pre className="text-xs overflow-auto mt-1 max-h-48 p-2 bg-white dark:bg-black rounded">
                    {JSON.stringify(part.output, null, 2)}
                  </pre>
                )}
              </div>
            );
          }
        }
        return null;
      })}

      {/* 3. Tool Invocations */}
      {toolInvocations.map((tool: any, idx: number) => {
        if (tool.state === 'call' || tool.state === 'partial-call') {
          return (
            <div
              key={`legacy-tool-call-${idx}`}
              className="bg-blue-50 dark:bg-blue-950 border border-blue-200 dark:border-blue-800 rounded px-3 py-2 text-xs"
            >
              <p className="font-semibold text-blue-700 dark:text-blue-300 flex items-center gap-1">
                <Zap className="h-3 w-3" /> Calling: {tool.toolName}
              </p>
            </div>
          );
        }
        if (tool.state === 'result') {
          return (
            <div
              key={`legacy-tool-result-${idx}`}
              className="bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800 rounded px-3 py-2 text-xs"
            >
              <p className="font-semibold text-green-700 dark:text-green-300 flex items-center gap-1">
                <CheckCircle className="h-3 w-3" /> {tool.toolName}
              </p>
              {typeof tool.result === 'string' ? (
                <p className="mt-1 whitespace-pre-wrap">{tool.result}</p>
              ) : (
                <pre className="text-xs overflow-auto mt-1 max-h-48 p-2 bg-white dark:bg-black rounded">
                  {JSON.stringify(tool.result, null, 2)}
                </pre>
              )}
            </div>
          );
        }
        return null;
      })}
    </div>
  );
});

MemoizedMessageContent.displayName = 'MemoizedMessageContent';

// Specialized component for the active streaming message to avoid re-rendering others
export function renderMessageContent(message: any): React.ReactNode {
  return <MemoizedMessageContent message={message} />;
}

const StreamingMessage = renderMessageContent;

export function ChatInterface({
  initialSessionId,
  initialMessages = [],
}: {
  initialSessionId?: string;
  initialMessages?: any[];
}) {
  const { settings, mounted } = useAISettings();
  const activeModel = AI_MODELS[settings.modelId] || AI_MODELS['gpt-4o-mini'];
  const [input, setInput] = useState('');

  const { messages, setMessages, sendMessage, status } = useChat({
    api: '/api/chat',
    id: initialSessionId,
    initialMessages,
    body: { modelId: settings.modelId },
  } as any);

  // Load messages from localStorage on mount if no initial messages
  useEffect(() => {
    if (initialMessages.length === 0) {
      const savedMessages = localStorage.getItem('wealthos-chat-history');
      if (savedMessages) {
        try {
          setMessages(JSON.parse(savedMessages));
        } catch (e) {
          console.error('Failed to load chat history:', e);
        }
      }
    }
  }, [initialMessages.length, setMessages]);

  // Use the new debounced persistence hook instead of manual useEffect
  useDebouncedChatPersistence(messages, status);

  const isBusy = status === 'streaming' || status === 'submitted';

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isBusy) return;
    console.log('[Chat UI] Sending message:', input);
    void sendMessage({ text: input });
    setInput('');
  };

  return (
    <div className="flex flex-col h-full rounded-xl border bg-card text-card-foreground shadow-lg overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b bg-muted/30">
        <div className="flex items-center gap-3">
          <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center">
            <Bot className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h3 className="text-sm font-semibold">WealthOS AI Advisor</h3>
            <p className="text-[10px] text-muted-foreground flex items-center gap-1">
              <Sparkles className="h-2.5 w-2.5 text-amber-500" />
              {mounted ? activeModel.label : 'Thinking...'}
            </p>
          </div>
        </div>
      </div>

      {/* Messages */}
      <ScrollArea className="flex-1 p-6">
        <div className="space-y-6 max-w-3xl mx-auto">
          {messages.length === 0 ? (
            <div className="flex flex-col h-[400px] items-center justify-center text-center space-y-4">
              <div className="p-4 rounded-full bg-primary/5 border border-primary/10">
                <Bot className="h-10 w-10 text-primary/40" />
              </div>
              <div>
                <h4 className="font-semibold text-lg">How can I help you today?</h4>
                <p className="text-sm text-muted-foreground max-w-sm mx-auto">
                  Ask me about your net worth trends, budget performance, or specific transactions. I can analyze your
                  financial data in real-time.
                </p>
              </div>
            </div>
          ) : (
            messages.map((m, index) => {
              if (!hasContent(m)) return null;

              const isLast = index === messages.length - 1;
              const isStreaming = isLast && status === 'streaming';

              return (
                <div key={m.id} className={`flex gap-4 ${m.role === 'user' ? 'flex-row-reverse' : 'flex-row'}`}>
                  <div
                    className={`flex h-9 w-9 shrink-0 select-none items-center justify-center rounded-xl border shadow-sm ${
                      m.role === 'user' ? 'bg-primary text-primary-foreground border-primary' : 'bg-muted'
                    }`}
                  >
                    {m.role === 'user' ? <User className="h-4.5 w-4.5" /> : <Bot className="h-4.5 w-4.5" />}
                  </div>
                  <div
                    className={`flex flex-col gap-2 rounded-2xl px-5 py-3.5 text-sm max-w-[85%] leading-relaxed overflow-x-auto ${
                      m.role === 'user'
                        ? 'bg-primary text-primary-foreground rounded-tr-none shadow-md'
                        : 'bg-muted rounded-tl-none border shadow-sm'
                    }`}
                  >
                    {isStreaming ? <StreamingMessage message={m} /> : <MemoizedMessageContent message={m} />}
                  </div>
                </div>
              );
            })
          )}
          {isBusy && (
            <div className="flex gap-4 flex-row items-center">
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border shadow-sm bg-muted animate-pulse">
                <Sparkles className="h-4.5 w-4.5 animate-pulse text-indigo-500" />
              </div>
              <div className="bg-muted rounded-2xl rounded-tl-none px-5 py-3.5 border border-dashed">
                <div className="flex gap-1.5 items-center">
                  <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce [animation-delay:-0.3s]" />
                  <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce [animation-delay:-0.15s]" />
                  <span className="h-1.5 w-1.5 rounded-full bg-primary/40 animate-bounce" />
                </div>
              </div>
            </div>
          )}
        </div>
      </ScrollArea>

      {/* Input Form */}
      <div className="p-6 border-t bg-card">
        <form onSubmit={handleFormSubmit} className="flex w-full items-end gap-3 max-w-3xl mx-auto">
          <div className="flex-1 relative">
            <textarea
              value={input}
              onChange={(e) => {
                const target = e.target;
                setInput(target.value);
                target.style.height = 'inherit';
                target.style.height = `${Math.min(target.scrollHeight, 200)}px`;
              }}
              placeholder="Message WealthOS AI..."
              className="w-full rounded-2xl border border-input bg-muted/30 px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary transition-all resize-none min-h-[48px] max-h-[200px]"
              disabled={isBusy}
              rows={1}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  handleFormSubmit(e as any);
                }
              }}
            />
          </div>
          <Button
            type="submit"
            size="icon"
            className="h-11 w-11 rounded-full shadow-lg transition-transform active:scale-95"
            disabled={isBusy || !input.trim()}
          >
            <Send className="h-5 w-5" />
          </Button>
        </form>
        <p className="text-[10px] text-center text-muted-foreground mt-3 uppercase tracking-tighter">
          Analysis computed on live Google Sheet data
        </p>
      </div>
    </div>
  );
}
