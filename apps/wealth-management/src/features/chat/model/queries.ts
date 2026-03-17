/**
 * Chat domain queries - read-only operations
 */

import { ChatMessage, SuggestionItem } from "./types";
import { AI_MODELS } from "@wealth-management/ai";

const STORAGE_KEY = "wealthos-chat-history";

/**
 * Generate a unique message ID
 */
export function generateMessageId(): string {
  return Date.now().toString(36) + Math.random().toString(36).substring(2);
}

/**
 * Load chat history from localStorage
 * @return Array of saved chat messages
 */
export function loadChatHistory(): ChatMessage[] {
  if (typeof window === "undefined") return [];
  
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      if (Array.isArray(parsed)) {
        return parsed;
      }
    }
  } catch (error: unknown) {
    console.error("Failed to load chat history:", error);
  }
  
  return [];
}

/**
 * Get active model label
 * @param modelId - The AI model ID
 * @return Human-readable label for the model
 */
export function getActiveModelLabel(modelId: string): string {
  return AI_MODELS[modelId]?.label || modelId;
}

/**
 * Fetch chat suggestions based on context
 * @param modelId - The AI model ID
 * @param context - Optional context for suggestions
 * @return Array of suggestions
 */
export async function fetchSuggestions(
  modelId: string,
  context?: Record<string, unknown>
): Promise<SuggestionItem[]> {
  try {
    const response = await fetch("/api/chat/suggestions", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        modelId,
        context,
      }),
    });

    if (!response.ok) {
      throw new Error(`API error: ${response.statusText}`);
    }

    const data = (await response.json()) as { suggestions: SuggestionItem[] };
    return data.suggestions || [];
  } catch (error: unknown) {
    console.error("Failed to fetch suggestions:", error);
    // Return fallback suggestions
    return [
      {
        label: "Financial Review",
        prompt: "Give me a full financial review for this month.",
      },
      {
        label: "Net Worth",
        prompt: "What is my current net worth breakdown?",
      },
      {
        label: "Savings Tips",
        prompt: "How can I save more money this month?",
      },
    ];
  }
}
