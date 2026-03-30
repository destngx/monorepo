/**
 * Chat domain types
 */

export interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system" | "data";
  content?: string;
  createdAt?: Date;
  parts?: Array<{
    type: string;
    text?: string;
    [key: string]: unknown;
  }>;
  toolInvocations?: Array<{
    state: string;
    toolName: string;
    [key: string]: unknown;
  }>;
}

export interface ChatResponse {
  message: ChatMessage;
  error?: string;
}

export interface SuggestionItem {
  label: string;
  prompt: string;
}

export interface SuggestionsResponse {
  suggestions: SuggestionItem[];
}

export interface AIInsight {
  id: string;
  title: string;
  content: string;
  createdAt: Date;
  metadata?: Record<string, unknown>;
}
