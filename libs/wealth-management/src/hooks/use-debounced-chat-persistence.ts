import { useEffect, useRef } from 'react';

export function useDebouncedChatPersistence(
  messages: any[],
  status: string,
  storageKey = 'wealthos-chat-history',
) {
  const timeoutRef = useRef<NodeJS.Timeout | null>(null);

  useEffect(() => {
    // If we're currently streaming, we don't want to save on every token.
    // Instead, we wait until the streaming is done, or periodically if long-running.

    if (status === 'streaming') {
      // Clear any pending save
      if (timeoutRef.current) clearTimeout(timeoutRef.current);

      // Schedule a save for after the stream stays quiet for 2 seconds
      timeoutRef.current = setTimeout(() => {
        saveToStorage();
      }, 2000);

      return;
    }

    // If status is not streaming (e.g., 'ready', 'error'), save immediately
    saveToStorage();

    function saveToStorage() {
      if (messages.length === 0) return;
      try {
        localStorage.setItem(storageKey, JSON.stringify(messages));
      } catch (e) {
        console.error('Failed to save chat history:', e);
      }
    }

    return () => {
      if (timeoutRef.current) clearTimeout(timeoutRef.current);
    };
  }, [messages, status, storageKey]);
}
