'use client';

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { Bot, Sparkles } from 'lucide-react';
import { ChatMessages } from './chat-messages';
import { ChatInput } from './chat-input';
import { useAISettings } from '@/hooks/use-ai-settings';
import { AI_MODELS } from '@wealth-management/ai';
import { isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  createdAt?: Date;
}

const STORAGE_KEY = 'wealthos-chat-history';

function generateId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

export function ChatContainer(): React.ReactNode {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isMounted, setIsMounted] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);
  const { settings } = useAISettings();

  // Load messages from localStorage on mount
  useEffect(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        if (Array.isArray(parsed)) {
          setMessages(parsed);
        }
      }
    } catch (error) {
      const message = getErrorMessage(error);
      console.error('Failed to load chat history:', message);
    }
    setIsMounted(true);
  }, []);

  // Save messages to localStorage whenever they change
  useEffect(() => {
    if (isMounted && messages.length > 0) {
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(messages));
      } catch (error) {
        const message = getErrorMessage(error);
        console.error('Failed to save chat history:', message);
      }
    }
  }, [messages, isMounted]);

  const handleSendMessage = useCallback(
    async (text: string) => {
      // Add user message
      const userMessage: Message = {
        id: generateId(),
        role: 'user',
        content: text,
        createdAt: new Date(),
      };

      setMessages((prev) => [...prev, userMessage]);
      setIsLoading(true);

      // Create abort controller for this request
      abortControllerRef.current = new AbortController();

      try {
        // Prepare messages for API
        const apiMessages = messages.concat(userMessage).map(({ id: _id, ...msg }) => msg);

        const response = await fetch('/api/chat', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            messages: apiMessages,
            modelId: settings.modelId,
          }),
          signal: abortControllerRef.current.signal,
        });

        if (!response.ok) {
          throw new Error(`API error: ${response.statusText}`);
        }

        // Handle streaming response using standard AI SDK Data Stream format
        const reader = response.body?.getReader();
        if (!reader) {
          throw new Error('No response body');
        }

        let fullContent = '';
        const assistantMessage: Message = {
          id: generateId(),
          role: 'assistant',
          content: '',
          createdAt: new Date(),
        };

        setMessages((prev) => [...prev, assistantMessage]);

        const decoder = new TextDecoder();
        let buffer = '';

        // eslint-disable-next-line no-constant-condition
        while (true) {
          const { done: isDone, value } = await reader.read();
          if (isDone) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            if (!line.trim()) continue;

            let data = line;
            if (data.startsWith('data: ')) {
              data = data.slice(6);
            }

            if (data === '[DONE]') continue;

            try {
              const parsed = JSON.parse(data);

              // Handle text deltas in the UIMessageChunk format
              if (parsed.type === 'text-delta' && parsed.delta) {
                fullContent += parsed.delta;
                setMessages((prev) => {
                  const updated = [...prev];
                  const lastMessage = updated[updated.length - 1];
                  if (lastMessage && lastMessage.role === 'assistant') {
                    updated[updated.length - 1] = {
                      ...lastMessage,
                      content: fullContent,
                    };
                  }
                  return updated;
                });
              }
            } catch (e) {
              // Ignore non-JSON lines
            }
          }
        }
      } catch (error: any) {
        if (error.name === 'AbortError') {
          /* eslint-disable-next-line no-console */
          console.log('Request cancelled');
          return;
        }

        const message = getErrorMessage(error);
        /* eslint-disable-next-line no-console */
        console.error('Failed to get response:', message);

        // Add error message
        const errorMessage: Message = {
          id: generateId(),
          role: 'assistant',
          content: `Sorry, I encountered an error: ${error.message || 'Unknown error'}. Please try again.`,
          createdAt: new Date(),
        };

        setMessages((prev) => [...prev, errorMessage]);
      } finally {
        setIsLoading(false);
        abortControllerRef.current = null;
      }
    },
    [messages, settings.modelId],
  );

  if (!isMounted) {
    return null; // Prevent hydration mismatch
  }

  const activeModelLabel = AI_MODELS[settings.modelId]?.label || settings.modelId;

  return (
    <div className="flex flex-col h-full rounded-xl border bg-card shadow-lg overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-6 py-4 border-b bg-gradient-to-r from-primary/5 to-transparent">
        <div className="flex items-center gap-3">
          <div className="h-8 w-8 rounded-lg bg-primary/10 flex items-center justify-center">
            <Bot className="h-5 w-5 text-primary" />
          </div>
          <div>
            <h2 className="text-sm font-semibold">WealthOS AI Advisor</h2>
            <p className="text-xs text-muted-foreground flex items-center gap-1">
              <Sparkles className="h-3 w-3 text-amber-500" />
              {activeModelLabel}
            </p>
          </div>
        </div>
      </div>

      {/* Messages */}
      <ChatMessages messages={messages} isLoading={isLoading} />

      {/* Input */}
      <div className="flex-shrink-0 p-6 border-t bg-card">
        <ChatInput onSubmit={handleSendMessage} isLoading={isLoading} placeholder="Ask about your finances..." />
      </div>
    </div>
  );
}
