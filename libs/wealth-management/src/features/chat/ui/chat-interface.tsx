'use client';

import { useState, useEffect } from 'react';
import { useChat } from '@ai-sdk/react';
import { ScrollArea } from '@wealth-management/ui';
import { Bot, User, Sparkles } from 'lucide-react';
import { useAISettings, useDebouncedChatPersistence } from '../../../hooks';
import { AI_MODELS } from '@wealth-management/ai';
import { AppError, isAppError } from '../../../utils/errors';
import { MessageContent } from './components/MessageContent';
import { ChatHeader } from './components/ChatHeader';
import { ChatInputForm } from './components/ChatInputForm';

export function hasContent(message: any): boolean {
  if (Array.isArray(message.parts) && message.parts.length > 0) return true;
  if (message.toolInvocations && message.toolInvocations.length > 0) return true;
  if (message.content) return true;
  return message.role === 'user';
}

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

  useEffect(() => {
    if (initialMessages.length === 0) {
      const savedMessages = localStorage.getItem('wealthos-chat-history');
      if (savedMessages) {
        try {
          setMessages(JSON.parse(savedMessages));
        } catch (e) {
          if (isAppError(e)) {
            console.error('Failed to load chat history:', e.message);
          } else {
            const error = new AppError(e instanceof Error ? e.message : 'Failed to parse chat history');
            console.error('Failed to load chat history:', error.message);
          }
        }
      }
    }
  }, [initialMessages.length, setMessages]);

  useDebouncedChatPersistence(messages, status);

  const isBusy = status === 'streaming' || status === 'submitted';

  const handleFormSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isBusy) return;
    sendMessage({ text: input });
    setInput('');
  };

  return (
    <div className="flex flex-col h-full rounded-xl border bg-card text-card-foreground shadow-lg overflow-hidden">
      <ChatHeader mounted={mounted} activeModelLabel={activeModel.label} />

      <ScrollArea className="flex-1 p-6">
        <div className="space-y-6 max-w-3xl mx-auto">
          {messages.length === 0 ? (
            <div className="flex flex-col h-[400px] items-center justify-center text-center space-y-4">
              <div className="p-4 rounded-full bg-primary/5 border border-primary/10">
                <Bot className="h-10 w-10 text-primary/40" />
              </div>
              <div className="space-y-2">
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
                    <MessageContent message={m} />
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

      <ChatInputForm 
        input={input} 
        setInput={setInput} 
        onSubmit={handleFormSubmit} 
        isBusy={isBusy} 
      />
    </div>
  );
}
