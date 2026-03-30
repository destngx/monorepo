/**
 * Chat domain mutations - write operations
 */

import { ChatMessage } from './types';
import { ChatError, StorageError } from '../../../utils/errors';

const STORAGE_KEY = 'wealthos-chat-history';

/**
 * Save chat history to localStorage
 */
export function saveChatHistory(messages: ChatMessage[]): boolean {
  if (typeof window === 'undefined') return false;

  try {
    if (messages.length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(messages));
    }
    return true;
  } catch (error) {
    const storageError = new StorageError('Failed to save chat history', {
      context: { action: 'saveChatHistory', error },
      userMessage: 'Unable to save chat history. Your conversation may not persist.',
    });
    console.error(storageError.message, storageError);
    return false;
  }
}

/**
 * Clear chat history
 */
export function clearChatHistory(): boolean {
  if (typeof window === 'undefined') return false;

  try {
    localStorage.removeItem(STORAGE_KEY);
    return true;
  } catch (error) {
    const storageError = new StorageError('Failed to clear chat history', {
      context: { action: 'clearChatHistory', error },
      userMessage: 'Unable to clear chat history.',
    });
    console.error(storageError.message, storageError);
    return false;
  }
}

/**
 * Send chat message to API
 */
export async function sendChatMessage(
  messages: ChatMessage[],
  modelId: string,
  signal?: AbortSignal,
): Promise<ReadableStream<Uint8Array>> {
  const apiMessages = messages.map(({ id, ...msg }) => msg);

  const response = await fetch('/api/chat', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      messages: apiMessages,
      modelId,
    }),
    signal,
  });

  if (!response.ok) {
    throw new ChatError(`Chat API error: ${response.statusText}`, {
      context: { endpoint: '/api/chat', statusCode: response.status },
      userMessage: 'Failed to send message. Please try again.',
    });
  }

  if (!response.body) {
    throw new ChatError('No response body from chat API', {
      context: { endpoint: '/api/chat' },
      userMessage: 'Invalid response from server.',
    });
  }

  return response.body;
}
