'use client';

import React, { useState, useRef, useCallback } from 'react';
import { Send, Loader2 } from 'lucide-react';
import { Button } from '-management/ui/button';
import { ChatError, isAppError } from '../../../utils/errors';

interface ChatInputProps {
  onSubmit: (message: string) => Promise<void>;
  isLoading?: boolean;
  placeholder?: string;
}

export function ChatInput({ onSubmit, isLoading = false, placeholder = 'Message WealthOS AI...' }: ChatInputProps) {
  const [input, setInput] = useState('');
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!input.trim() || isLoading) return;

      const message = input.trim();
      setInput('');

      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }

      try {
        await onSubmit(message);
      } catch (error) {
        const chatError = isAppError(error)
          ? error
          : new ChatError('Failed to send message', {
              context: { error },
              userMessage: 'Unable to send your message. Please try again.',
            });

        console.error(chatError.message, chatError);
        setInput(message);
      }
    },
    [input, isLoading, onSubmit],
  );

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e as any);
    }
  };

  const handleChange = (e: React.ChangeEvent<HTMLTextAreaElement>) => {
    setInput(e.target.value);

    // Auto-resize textarea
    const textarea = e.target;
    textarea.style.height = 'auto';
    textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
  };

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-2">
      <div className="flex w-full items-end gap-3">
        <textarea
          ref={textareaRef}
          value={input}
          onChange={handleChange}
          onKeyDown={handleKeyDown}
          placeholder={placeholder}
          disabled={isLoading}
          rows={1}
          className="flex-1 rounded-lg border border-input bg-background px-4 py-3 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-primary/50 focus:border-primary disabled:opacity-50 disabled:cursor-not-allowed min-h-[44px] max-h-[200px]"
        />
        <Button type="submit" size="icon" disabled={isLoading || !input.trim()} className="h-10 w-10 shrink-0">
          {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
        </Button>
      </div>
      <p className="text-xs text-center text-muted-foreground">Powered by live financial data</p>
    </form>
  );
}
