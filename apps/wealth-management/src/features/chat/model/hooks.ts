/**
 * Chat domain hooks for React component state management
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { ChatMessage } from './types';
import { loadChatHistory, generateMessageId, fetchSuggestions } from './queries';
import { saveChatHistory, clearChatHistory, sendChatMessage } from './mutations';

/**
 * Hook to manage chat messages state and persistence
 */
export function useChatMessages() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [mounted, setMounted] = useState(false);

  // Load messages from localStorage on mount
  useEffect(() => {
    const saved = loadChatHistory();
    setMessages(saved);
    setMounted(true);
  }, []);

  // Save messages to localStorage whenever they change
  useEffect(() => {
    if (mounted && messages.length > 0) {
      saveChatHistory(messages);
    }
  }, [messages, mounted]);

  const addMessage = useCallback((role: 'user' | 'assistant', content: string) => {
    const message: ChatMessage = {
      id: generateMessageId(),
      role,
      content,
      createdAt: new Date(),
    };
    setMessages((prev) => [...prev, message]);
    return message;
  }, []);

  const updateLastMessage = useCallback((content: string) => {
    setMessages((prev) => {
      const updated = [...prev];
      const lastMessage = updated[updated.length - 1];
      if (lastMessage && lastMessage.role === 'assistant') {
        updated[updated.length - 1] = {
          ...lastMessage,
          content,
        };
      }
      return updated;
    });
  }, []);

  const clear = useCallback(() => {
    clearChatHistory();
    setMessages([]);
  }, []);

  return {
    messages,
    mounted,
    addMessage,
    updateLastMessage,
    clear,
  };
}

/**
 * Hook to manage chat API streaming
 */
export function useChatStream() {
  const [isLoading, setIsLoading] = useState(false);
  const abortControllerRef = useRef<AbortController | null>(null);

  const streamMessage = useCallback(
    async (messages: ChatMessage[], modelId: string, onChunk: (content: string) => void) => {
      setIsLoading(true);
      abortControllerRef.current = new AbortController();

      try {
        const body = await sendChatMessage(messages, modelId, abortControllerRef.current.signal);

        const reader = body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let fullContent = '';

        let reading = true;
        while (reading) {
          const { done: isDone, value } = await reader.read();
          if (isDone) {
            reading = false;
            break;
          }

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
                onChunk(fullContent);
              }
            } catch (e) {
              // Ignore non-JSON lines
            }
          }
        }
      } finally {
        setIsLoading(false);
        abortControllerRef.current = null;
      }
    },
    [],
  );

  const cancel = useCallback(() => {
    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
      abortControllerRef.current = null;
      setIsLoading(false);
    }
  }, []);

  return {
    isLoading,
    streamMessage,
    cancel,
  };
}

/**
 * Hook to manage chat suggestions
 * @param modelId - The AI model ID
 * @param context - Optional context for suggestions
 * @return Object with suggestions and loading state
 */
export function useChatSuggestions(modelId: string, context?: Record<string, unknown>) {
  const [suggestions, setSuggestions] = useState<Array<{ label: string; prompt: string }>>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    setIsLoading(true);
    void fetchSuggestions(modelId, context)
      .then(setSuggestions)
      .finally(() => setIsLoading(false));
  }, [modelId, context]);

  return { suggestions, isLoading };
}
