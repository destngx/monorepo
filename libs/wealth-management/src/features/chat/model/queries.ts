/**
 * Chat domain queries - read-only operations
 */

import { ChatMessage, SuggestionItem } from './types';
import { AI_MODELS } from '@wealth-management/ai';
import { ChatError, StorageError } from '../../../utils/errors';

const STORAGE_KEY = 'wealthos-chat-history';

/**
 * Generate a unique message ID
 */
export function generateMessageId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substr(2);
}

/**
 * Load chat history from localStorage
 */
export function loadChatHistory(): ChatMessage[] {
  if (typeof window === 'undefined') return [];

  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      if (Array.isArray(parsed)) {
        return parsed;
      }
    }
  } catch (error) {
    const storageError = new StorageError('Failed to load chat history', {
      context: { action: 'loadChatHistory', error },
      userMessage: 'Unable to load chat history.',
    });
    console.error(storageError.message, storageError);
  }

  return [];
}

/**
 * Get active model label
 */
export function getActiveModelLabel(modelId: string): string {
  return AI_MODELS[modelId]?.label || modelId;
}

/**
 * Fetch chat suggestions based on context
 */
export async function fetchSuggestions(modelId: string, context?: Record<string, any>): Promise<SuggestionItem[]> {
  try {
    const response = await fetch('/api/chat/suggestions', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        modelId,
        context,
      }),
    });

    if (!response.ok) {
      throw new ChatError(`Chat suggestions API error: ${response.statusText}`, {
        context: { endpoint: '/api/chat/suggestions', statusCode: response.status },
        userMessage: 'Failed to fetch suggestions.',
      });
    }

    const data = await response.json();
    return data.suggestions || [];
  } catch (error) {
    const chatError =
      error instanceof ChatError
        ? error
        : new ChatError('Failed to fetch chat suggestions', {
            context: { action: 'fetchSuggestions', error },
            userMessage: 'Unable to fetch suggestions. Using defaults.',
          });
    console.error(chatError.message, chatError);
    // Return fallback suggestions
    return [
      {
        label: 'Financial Review',
        prompt: 'Give me a full financial review for this month.',
      },
      {
        label: 'Net Worth',
        prompt: 'What is my current net worth breakdown?',
      },
      {
        label: 'Savings Tips',
        prompt: 'How can I save more money this month?',
      },
    ];
  }
}
