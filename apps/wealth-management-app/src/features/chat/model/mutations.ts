/**
 * Chat domain mutations - write operations
 */

import { ChatMessage } from "./types";

const STORAGE_KEY = "wealthos-chat-history";

/**
 * Save chat history to localStorage
 */
export function saveChatHistory(messages: ChatMessage[]): boolean {
  if (typeof window === "undefined") return false;
  
  try {
    if (messages.length > 0) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(messages));
    }
    return true;
  } catch (error) {
    console.error("Failed to save chat history:", error);
    return false;
  }
}

/**
 * Clear chat history
 */
export function clearChatHistory(): boolean {
  if (typeof window === "undefined") return false;
  
  try {
    localStorage.removeItem(STORAGE_KEY);
    return true;
  } catch (error) {
    console.error("Failed to clear chat history:", error);
    return false;
  }
}

/**
 * Send chat message to API
 */
export async function sendChatMessage(
  messages: ChatMessage[],
  modelId: string,
  signal?: AbortSignal
): Promise<ReadableStream<Uint8Array>> {
  const apiMessages = messages.map(({ id, ...msg }) => msg);

  const response = await fetch("/api/chat", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      messages: apiMessages,
      modelId,
    }),
    signal,
  });

  if (!response.ok) {
    throw new Error(`API error: ${response.statusText}`);
  }

  if (!response.body) {
    throw new Error("No response body");
  }

  return response.body;
}
